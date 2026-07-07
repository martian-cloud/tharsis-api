package workspace

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/aws/smithy-go/ptr"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/commands"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/metric"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

const (
	// maxSleepIntervalSeconds is the max amount of time that the scheduler will sleep
	maxSleepIntervalSeconds = 600
	// minSleepIntervalSeconds is the min amount of time that the schedule will sleep
	minSleepIntervalSeconds = 300
	// workspaceBatchSize is the number of workspaces to check in a single batch
	workspaceBatchSize = 100
)

// pageSleepInterval is the delay between paginated workspace batches within a single
// scheduler pass, so we page through all workspaces quickly rather than waiting the full
// scheduler interval between pages. It's a var so tests can shorten it.
var pageSleepInterval = 1 * time.Second

var (
	assessmentSchedulerAttempts        = metric.NewCounter("workspace_assessment_scheduler_attempts", "Amount of assessment scheduler attempts.")
	assessmentSchedulerAssessmentCount = metric.NewCounter("workspace_assessment_scheduler_assessment_count", "Total number of workspaces assessed")
	assessmentSchedulerAttemptDuration = metric.NewHistogram("workspace_assessment_scheduler_attempt_duration", "Amount of time to create workspace assessments during a single attempt.", 1, 2, 8)
)

// AssessmentScheduler is a service for detecting assessments in a workspace.
type AssessmentScheduler struct {
	dbClient                  *db.Client
	logger                    logger.Logger
	cmdProcessor              engine.CmdProcessor
	cmdFactory                *commands.Factory
	inheritedSettingsResolver namespace.InheritedSettingResolver
	maintenanceMonitor        maintenance.Monitor
	assessmentMinInterval     time.Duration
	assessmentRunLimit        int
}

// NewAssessmentScheduler creates a new assessment scheduler.
func NewAssessmentScheduler(
	dbClient *db.Client,
	logger logger.Logger,
	cmdProcessor engine.CmdProcessor,
	cmdFactory *commands.Factory,
	inheritedSettingsResolver namespace.InheritedSettingResolver,
	maintenanceMonitor maintenance.Monitor,
	assessmentMinInterval time.Duration,
	assessmentRunLimit int,
) *AssessmentScheduler {
	return &AssessmentScheduler{
		dbClient:                  dbClient,
		logger:                    logger,
		cmdProcessor:              cmdProcessor,
		cmdFactory:                cmdFactory,
		inheritedSettingsResolver: inheritedSettingsResolver,
		maintenanceMonitor:        maintenanceMonitor,
		assessmentMinInterval:     assessmentMinInterval,
		assessmentRunLimit:        assessmentRunLimit,
	}
}

// Start starts the assessment scheduler.
func (a *AssessmentScheduler) Start(ctx context.Context) {
	a.logger.Info("workspace assessment scheduler started")

	go func() {
		for {
			assessmentSchedulerAttempts.Inc()

			// Use randomization for the sleep duration to prevent all nodes from checking the DB at the same time
			sleep := time.Duration(rand.IntN(maxSleepIntervalSeconds-minSleepIntervalSeconds) + minSleepIntervalSeconds)

			select {
			case <-time.After(sleep * time.Second):
				if err := a.execute(ctx); err != nil {
					a.logger.Error(err)
				}
			case <-ctx.Done():
				a.logger.Info("workspace assessment scheduler stopped")
				return
			}
		}
	}()
}

func (a *AssessmentScheduler) execute(ctx context.Context) error {
	// Check if we're in maintenance mode
	inMaintenance, err := a.maintenanceMonitor.InMaintenanceMode(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to check for maintenance mode in assessment scheduler")
	}
	if inMaintenance {
		return nil
	}

	// Paginate through all workspaces in a single pass, sleeping briefly between pages
	// rather than waiting the full scheduler interval between each page.
	var cursor *string
	for {
		nextCursor, limitExceeded, err := a.checkWorkspaces(ctx, cursor)
		if err != nil {
			return errors.Wrap(err, "failed to check workspaces in assessment scheduler")
		}

		// Stop the pass and wait for the next scheduled run when the in-progress assessment
		// limit is reached, or when there are no more pages of workspaces to check.
		if limitExceeded || nextCursor == nil {
			return nil
		}
		cursor = nextCursor

		select {
		case <-time.After(pageSleepInterval):
		case <-ctx.Done():
			return nil
		}
	}
}

func (a *AssessmentScheduler) checkWorkspaces(ctx context.Context, cursor *string) (*string, bool, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		assessmentSchedulerAttemptDuration.Observe(float64(duration.Seconds()))
	}()

	var nextCursor *string

	// Get next batch of workspaces to check.
	//
	// We order by full path rather than least-recently-assessed. When the global limit
	// binds this favors alphabetically-early workspaces, but the bias is mild in practice:
	// passes run every few seconds and the MinDurationSinceLastAssessment filter keeps
	// not-yet-assessed workspaces eligible, so unreached workspaces are picked up on a
	// subsequent pass.
	workspaceSort := db.WorkspaceSortableFieldFullPathAsc
	workspaces, err := a.dbClient.Workspaces.GetWorkspaces(ctx, &db.GetWorkspacesInput{
		Sort: &workspaceSort,
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(workspaceBatchSize), // Get the next batch of workspaces
			After: cursor,
		},
		Filter: &db.WorkspaceFilter{
			MinDurationSinceLastAssessment: &a.assessmentMinInterval,
			Locked:                         ptr.Bool(false),
			HasStateVersion:                ptr.Bool(true),
		},
	})
	if err != nil {
		return nil, false, fmt.Errorf("failed to get workspaces: %w", err)
	}

	settingCache := map[string]bool{}

	for _, workspace := range workspaces.Workspaces {
		enabled, err := a.isDriftDetectionEnabled(ctx, &workspace, settingCache)
		if err != nil {
			return nil, false, fmt.Errorf("failed to get drift detection enabled %w", err)
		}
		if enabled {
			limitExceeded, err := a.startWorkspaceAssessment(ctx, &workspace)
			if err != nil {
				a.logger.Errorf("failed to run workspace assessment for workspace %q: %v", workspace.FullPath, err)
			}
			if limitExceeded {
				// Stop the pass; the next scheduled run resumes from the first page.
				return nil, true, nil
			}
		}
		nextCursor, err = workspaces.PageInfo.Cursor(&workspace)
		if err != nil {
			return nil, false, errors.Wrap(err, "failed to get next cursor in assessment scheduler")
		}
	}

	if !workspaces.PageInfo.HasNextPage {
		nextCursor = nil
	}

	return nextCursor, false, nil
}

func (a *AssessmentScheduler) isDriftDetectionEnabled(ctx context.Context, workspace *models.Workspace, cache map[string]bool) (bool, error) {
	if workspace.EnableDriftDetection != nil {
		return *workspace.EnableDriftDetection, nil
	}

	// Check if the setting is already cached for parent group path
	groupPath := workspace.GetGroupPath()
	if cachedValue, ok := cache[groupPath]; ok {
		return cachedValue, nil
	}

	setting, err := a.inheritedSettingsResolver.GetDriftDetectionEnabled(ctx, workspace)
	if err != nil {
		return false, fmt.Errorf("failed to get drift detection enabled %w", err)
	}

	// Cache under the same key we read by (the workspace's group path). Inherited settings
	// resolve identically for all workspaces sharing a group path, so this dedupes resolver
	// calls for sibling workspaces within a pass.
	cache[groupPath] = setting.Value

	return setting.Value, nil
}

// countInProgressAssessments returns the number of assessments currently in progress
// (stale ones excluded by the filter). The scheduler reads this once per pass to derive
// how many new assessment runs it may create.
func (a *AssessmentScheduler) countInProgressAssessments(ctx context.Context) (int, error) {
	response, err := a.dbClient.WorkspaceAssessments.GetWorkspaceAssessments(ctx, &db.GetWorkspaceAssessmentsInput{
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
		Filter: &db.WorkspaceAssessmentFilter{
			InProgress: ptr.Bool(true),
		},
	})
	if err != nil {
		return 0, err
	}

	totalCount, err := response.PageInfo.TotalCount(ctx)
	if err != nil {
		return 0, err
	}

	return int(totalCount), nil
}

// startWorkspaceAssessment creates an assessment run for the workspace if one is due. It
// returns true when the in-progress assessment limit has been reached, signaling the caller
// to stop the pass and wait for the next scheduled run.
func (a *AssessmentScheduler) startWorkspaceAssessment(ctx context.Context, workspace *models.Workspace) (bool, error) {
	assessment, err := a.dbClient.WorkspaceAssessments.GetWorkspaceAssessmentByWorkspaceID(ctx, workspace.Metadata.ID)
	if err != nil {
		return false, fmt.Errorf("failed to get workspace assessment for workspace %q: %w", workspace.FullPath, err)
	}

	var latestAssessmentVersion *int
	if assessment != nil {
		// Check if the assessment satisfies the minimum interval since it may have been
		// started manually or by another instance. A stale in-progress assessment (its run
		// was abandoned before completing) is restarted even within the minimum interval.
		if !assessment.IsStaleInProgress() && time.Since(assessment.StartedAtTimestamp) < a.assessmentMinInterval {
			return false, nil
		}
		latestAssessmentVersion = &assessment.Metadata.Version
	}

	// Re-read the live in-progress count (which reflects assessments created by all API
	// instances) before creating, so the global limit is honored rather than a stale
	// per-pass snapshot. Not-due workspaces return above without reaching this, and the pass
	// stops as soon as the limit is hit, so the number of count queries per pass is bounded
	// by the limit rather than the workspace count.
	inProgress, err := a.countInProgressAssessments(ctx)
	if err != nil {
		return false, err
	}
	if inProgress >= a.assessmentRunLimit {
		// Limit reached: stop the pass and wait for the next scheduled run.
		return true, nil
	}

	// Create the assessment run by invoking the command directly; the command processor
	// owns its own transaction. Assessments are system-initiated, so the run's subject is
	// the system caller and no caller is required on the context.
	cmd := a.cmdFactory.NewCreateAssessmentRun(&commands.CreateAssessmentRunInput{
		Subject:                 auth.System,
		WorkspaceID:             workspace.Metadata.ID,
		LatestAssessmentVersion: latestAssessmentVersion,
		// Scheduler-triggered assessments run automatically and frequently, so suppress
		// the run-creation activity event to avoid flooding the activity feed.
		SkipActivityEvent: true,
	})
	if err = a.cmdProcessor.ProcessCommand(ctx, cmd); err != nil {
		switch errors.ErrorCode(err) {
		case errors.EOptimisticLock, errors.EConflict:
			// Both are expected when another instance assessed the same workspace
			// concurrently (optimistic lock) or an assessment is already in progress
			// (conflict); ignore them silently. All other errors are surfaced so the
			// caller can log why the assessment failed.
			return false, nil
		}
		return false, err
	}

	assessmentSchedulerAssessmentCount.Inc()

	a.logger.Infof("assessment scheduler created assessment run for workspace %q", workspace.FullPath)

	return false, nil
}
