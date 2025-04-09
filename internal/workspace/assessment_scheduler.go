package workspace

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/aws/smithy-go/ptr"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/metric"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
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

var (
	assessmentSchedulerAttempts        = metric.NewCounter("workspace_assessment_scheduler_attempts", "Amount of assessment scheduler attempts.")
	assessmentSchedulerAssessmentCount = metric.NewCounter("workspace_assessment_scheduler_assessment_count", "Total number of workspaces assessed")
	assessmentSchedulerAttemptDuration = metric.NewHistogram("workspace_assessment_scheduler_attempt_duration", "Amount of time to create workspace assessments during a single attempt.", 1, 2, 8)
)

// AssessmentScheduler is a service for detecting assessments in a workspace.
type AssessmentScheduler struct {
	dbClient                  *db.Client
	logger                    logger.Logger
	runService                run.Service
	inheritedSettingsResolver namespace.InheritedSettingResolver
	maintenanceMonitor        maintenance.Monitor
	assessmentMinInterval     time.Duration
	assessmentRunLimit        int
}

// NewAssessmentScheduler creates a new assessment scheduler.
func NewAssessmentScheduler(
	dbClient *db.Client,
	logger logger.Logger,
	runService run.Service,
	inheritedSettingsResolver namespace.InheritedSettingResolver,
	maintenanceMonitor maintenance.Monitor,
	assessmentMinInterval time.Duration,
	assessmentRunLimit int,
) *AssessmentScheduler {
	return &AssessmentScheduler{
		dbClient:                  dbClient,
		logger:                    logger,
		runService:                runService,
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
		// This cursor is used to paginate through the workspaces
		var cursor *string

		for {
			assessmentSchedulerAttempts.Inc()

			// Use randomization for the sleep duration to prevent all nodes from checking the DB at the same time
			sleep := time.Duration(rand.IntN(maxSleepIntervalSeconds-minSleepIntervalSeconds) + minSleepIntervalSeconds)

			select {
			case <-time.After(sleep * time.Second):
				nextCursor, err := a.execute(ctx, cursor)
				if err != nil {
					a.logger.Error(err)
					continue
				}
				cursor = nextCursor
			case <-ctx.Done():
				a.logger.Info("workspace assessment scheduler stopped")
				return
			}
		}
	}()
}

func (a *AssessmentScheduler) execute(ctx context.Context, cursor *string) (*string, error) {
	// Check if we're in maintenance mode
	inMaintenance, err := a.maintenanceMonitor.InMaintenanceMode(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check for maintenance mode in assessment scheduler")
	}
	if inMaintenance {
		return cursor, nil
	}

	nextCursor, err := a.checkWorkspaces(ctx, cursor)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check workspaces in assessment scheduler")
	}
	return nextCursor, nil
}

func (a *AssessmentScheduler) checkWorkspaces(ctx context.Context, cursor *string) (*string, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		assessmentSchedulerAttemptDuration.Observe(float64(duration.Seconds()))
	}()

	var nextCursor *string

	// Get next batch of workspaces to check
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
		return nil, fmt.Errorf("failed to get workspaces: %w", err)
	}

	settingCache := map[string]bool{}

	for _, workspace := range workspaces.Workspaces {
		enabled, err := a.isDriftDetectionEnabled(ctx, &workspace, settingCache)
		if err != nil {
			return nil, fmt.Errorf("failed to get drift detection enabled %w", err)
		}
		if enabled {
			limitExceeded, err := a.startWorkspaceAssessment(ctx, &workspace)
			if err != nil {
				a.logger.Errorf("failed to run workspace assessment for workspace %q: %v", workspace.FullPath, err)
			}
			if limitExceeded {
				// If the limit is exceeded, we stop checking workspaces
				return nextCursor, nil
			}
		}
		nextCursor, err = workspaces.PageInfo.Cursor(&workspace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get next cursor in assessment scheduler")
		}
	}

	if !workspaces.PageInfo.HasNextPage {
		nextCursor = nil
	}

	return nextCursor, nil
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

	// Add setting to cache
	cache[setting.NamespacePath] = setting.Value

	return setting.Value, nil
}

func (a *AssessmentScheduler) checkInProgressAssessmentLimit(ctx context.Context, limit int) (int32, bool, error) {
	response, err := a.dbClient.WorkspaceAssessments.GetWorkspaceAssessments(ctx, &db.GetWorkspaceAssessmentsInput{
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
		Filter: &db.WorkspaceAssessmentFilter{
			InProgress: ptr.Bool(true),
		},
	})
	if err != nil {
		return 0, false, err
	}
	return response.PageInfo.TotalCount, int(response.PageInfo.TotalCount) < limit, nil
}

func (a *AssessmentScheduler) startWorkspaceAssessment(ctx context.Context, workspace *models.Workspace) (bool, error) {
	assessment, err := a.dbClient.WorkspaceAssessments.GetWorkspaceAssessmentByWorkspaceID(ctx, workspace.Metadata.ID)
	if err != nil {
		return false, fmt.Errorf("failed to get workspace assessment for workspace %q: %w", workspace.FullPath, err)
	}

	var latestAssessmentVersion *int
	if assessment != nil {
		// Check if the assessment satisfies the minimum interval since it may have been started manually or by another instance
		if time.Since(assessment.StartedAtTimestamp) < a.assessmentMinInterval {
			return false, nil
		}
		latestAssessmentVersion = &assessment.Metadata.Version
	}

	txContext, err := a.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return false, err
	}
	defer func() {
		if txErr := a.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			a.logger.Errorf("failed to rollback tx for startWorkspaceAssessment: %v", txErr)
		}
	}()

	// Create assessment run
	_, err = a.runService.CreateAssessmentRunForWorkspace(auth.WithCaller(txContext, &auth.SystemCaller{}), &run.CreateAssessmentRunForWorkspaceInput{
		WorkspaceID:             workspace.Metadata.ID,
		LatestAssessmentVersion: latestAssessmentVersion,
	})
	if err != nil {
		if errors.ErrorCode(err) != errors.EInternal {
			// An error is only returned for internal server errors since other error types are expected
			return false, nil
		}
		return false, err
	}

	// Check if we're still under or equal to the limit after creating a new assessment run
	_, underLimit, err := a.checkInProgressAssessmentLimit(txContext, a.assessmentRunLimit+1)
	if err != nil {
		return false, err
	}

	if !underLimit {
		// Return nil to rollback the transaction
		return true, nil
	}

	assessmentSchedulerAssessmentCount.Inc()

	a.logger.Infof("assessment scheduler created assessment run for workspace %q", workspace.FullPath)

	return false, a.dbClient.Transactions.CommitTx(txContext)
}
