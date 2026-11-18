package workspace

import (
	"context"
	"fmt"
	"math/bits"
	"math/rand/v2"
	"time"

	"github.com/aws/smithy-go/ptr"
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
	// assessmentBackoffMaxMultiplier caps exponential backoff at 16x the base assessment
	// interval, so a workspace that keeps failing is still reconsidered periodically rather than
	// drifting further and further from being checked at all.
	assessmentBackoffMaxMultiplier = 16
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
	// passes run every few seconds and the AssessmentDue filter keeps not-yet-assessed
	// workspaces eligible, so unreached workspaces are picked up on a subsequent pass.
	workspaceSort := db.WorkspaceSortableFieldFullPathAsc
	workspaces, err := a.dbClient.Workspaces.GetWorkspaces(ctx, &db.GetWorkspacesInput{
		Sort: &workspaceSort,
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(workspaceBatchSize), // Get the next batch of workspaces
			After: cursor,
		},
		Filter: &db.WorkspaceFilter{
			AssessmentDue:   ptr.Bool(true),
			Locked:          ptr.Bool(false),
			HasStateVersion: ptr.Bool(true),
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
			limitExceeded, expectedVersion, err := a.startWorkspaceAssessment(ctx, &workspace)
			if err != nil {
				a.handleAssessmentError(ctx, &workspace, expectedVersion, err)
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

// assessmentErrorDisposition says how the scheduler should treat an error returned while
// creating a workspace's assessment run.
type assessmentErrorDisposition int

const (
	// assessmentErrorIgnore is a benign race with another API instance. Nothing is logged and
	// nothing is recorded, because the condition resolves itself on the next pass.
	assessmentErrorIgnore assessmentErrorDisposition = iota

	// assessmentErrorRecord is an expected, unfixable condition: a renamed or deleted module, an
	// expired credential, or a managed identity rule rejecting a system-initiated run. Recorded
	// on the assessment so the owner knows why drift detection stopped; logged at INFO because
	// it is a normal, anticipated part of processing, not something that needs attention.
	assessmentErrorRecord

	// assessmentErrorUnexpected is anything else, including uncoded errors, which ErrorCode
	// reports as EInternal. These stay at ERROR because they indicate a fault in Tharsis.
	assessmentErrorUnexpected
)

// classifyAssessmentError maps an error from creating an assessment run to its disposition. It
// is only meaningful for a non-nil error; a nil error classifies as unexpected rather than
// silently looking benign, so a caller that forgets to check cannot hide a bug.
func classifyAssessmentError(err error) assessmentErrorDisposition {
	switch errors.ErrorCode(err) {
	case errors.EOptimisticLock, errors.EConflict:
		// Another instance assessed the same workspace concurrently (optimistic lock) or an
		// assessment is already in progress (conflict).
		return assessmentErrorIgnore
	case errors.ENotFound, errors.EInvalid, errors.EForbidden:
		// ENotFound: the registry module is gone, or the source run or its state version was
		// removed. EInvalid: the module source will not parse, or an external registry rejected
		// the run's stored credential. EForbidden: a managed identity rule the system caller
		// cannot satisfy.
		return assessmentErrorRecord
	default:
		return assessmentErrorUnexpected
	}
}

// handleAssessmentError dispatches an error from startWorkspaceAssessment according to its
// disposition. It never returns an error, because one workspace's failure must not stop the pass.
// expectedVersion is whichever assessment version startWorkspaceAssessment read before this
// attempt (nil if none existed); it is threaded through to recordAssessmentFailure so a stale
// failure cannot overwrite a newer attempt.
func (a *AssessmentScheduler) handleAssessmentError(ctx context.Context, workspace *models.Workspace, expectedVersion *int, err error) {
	switch classifyAssessmentError(err) {
	case assessmentErrorIgnore:
		// Another instance already assessed this workspace, or an assessment is still running.
		// Expected under normal concurrency, so there is nothing to report.
	case assessmentErrorRecord:
		a.logger.Infof("assessment scheduler could not assess workspace %q: %v", workspace.FullPath, err)
		a.recordAssessmentFailure(ctx, workspace, expectedVersion, err.Error())
	default:
		a.logger.Errorf("failed to run workspace assessment for workspace %q: %v", workspace.FullPath, err)
	}
}

// assessmentBackoffInterval returns how long to wait before reconsidering a workspace after
// consecutiveFailures consecutive failures: the base interval on the first, doubling each
// further failure, capped at assessmentBackoffMaxMultiplier times the base. These conditions
// rarely resolve on their own, so a fixed interval would retry forever; backing off spends
// fewer cycles on a workspace that keeps failing. This only throttles the scheduler — "Run
// Drift Detection" still bypasses it.
func assessmentBackoffInterval(base time.Duration, consecutiveFailures int) time.Duration {
	if consecutiveFailures <= 1 {
		return base
	}

	// consecutiveFailures=2 -> 2x, 3 -> 4x, 4 -> 8x, 5+ -> capped at assessmentBackoffMaxMultiplier.
	shift := consecutiveFailures - 1
	maxShift := bits.Len(uint(assessmentBackoffMaxMultiplier)) - 1
	if shift > maxShift {
		shift = maxShift
	}

	return base * time.Duration(1<<uint(shift))
}

// recordAssessmentFailure stores why the workspace could not be assessed, and sets
// AvailableAtTimestamp so it drops out of the candidate set until backoff elapses. That
// suppression stops an unactionable condition from retrying every pass; escalating backoff
// stops it from retrying at a fixed interval forever. A later apply clears the record via the
// run handler; creating a run again resets it directly (see
// CreateAssessmentRun.upsertAssessment).
//
// expectedVersion is the assessment version startWorkspaceAssessment read before the attempt
// that just failed (nil if no row existed). Between that read and this one, another writer —
// another instance, or a user-triggered run — may have started a genuinely new, still-running
// attempt. Writing over it would misreport a run that is actually in progress as failed, and
// would let the scheduler re-select the workspace once backoff elapses even though that other
// run has not finished. So if the row has moved on since expectedVersion — version mismatch,
// or in progress and not stale — this is a no-op.
//
// This is best-effort, attempted once: a failure to record, including losing an optimistic-lock
// race, is logged and swallowed so the pass continues. The create branch below is the exception,
// treating another instance's row as success since it already recorded the same condition.
func (a *AssessmentScheduler) recordAssessmentFailure(ctx context.Context, workspace *models.Workspace, expectedVersion *int, message string) {
	truncated := models.TruncateAssessmentErrorMessage(message)

	assessment, err := a.dbClient.WorkspaceAssessments.GetWorkspaceAssessmentByWorkspaceID(ctx, workspace.Metadata.ID)
	if err != nil {
		a.logger.Errorf("failed to record assessment failure for workspace %q: %v", workspace.FullPath, err)
		return
	}

	now := time.Now().UTC()

	if assessment == nil {
		if expectedVersion != nil {
			// A row existed when the failed attempt started but is gone now. Assessments are
			// never deleted independently of their workspace, so this should not happen; treat
			// it the same as any other sign that the row has moved on, and skip.
			return
		}
		availableAt := now.Add(assessmentBackoffInterval(a.assessmentMinInterval, 1))
		newAssessment := &models.WorkspaceAssessment{
			WorkspaceID:          workspace.Metadata.ID,
			StartedAtTimestamp:   now,
			CompletedAtTimestamp: &now,
			ErrorMessage:         &truncated,
			AvailableAtTimestamp: &availableAt,
			ConsecutiveFailures:  1,
		}
		if err = newAssessment.Validate(); err != nil {
			a.logger.Errorf("failed to record assessment failure for workspace %q: %v", workspace.FullPath, err)
			return
		}
		if _, err = a.dbClient.WorkspaceAssessments.CreateWorkspaceAssessment(ctx, newAssessment); err != nil && errors.ErrorCode(err) != errors.EConflict {
			// EConflict means another instance created the record first; it hit the same
			// condition, so the reason is already recorded and there is nothing left to do.
			a.logger.Errorf("failed to record assessment failure for workspace %q: %v", workspace.FullPath, err)
		}
		return
	}

	// The row has moved on since expectedVersion: either another writer already changed it
	// (version mismatch — no row read at all also counts, since expectedVersion is nil then),
	// or it is now a genuinely new, still-running attempt (in progress and not stale). Either
	// way, the failure above does not describe the row's current state, so leave it alone.
	if expectedVersion == nil || assessment.Metadata.Version != *expectedVersion ||
		(assessment.CompletedAtTimestamp == nil && !assessment.IsStaleInProgress()) {
		return
	}

	// HasDrift is deliberately left alone. This attempt produced no fresh verdict, and
	// reporting a drifted workspace as clean because the assessment could not run would be a
	// false negative — the same reasoning the assessment run handler applies to a failed run.
	// Notification stays off because no new verdict was reached.
	assessment.RequiresNotification = false
	assessment.StartedAtTimestamp = now
	assessment.CompletedAtTimestamp = &now
	assessment.ErrorMessage = &truncated
	assessment.ConsecutiveFailures++
	availableAt := now.Add(assessmentBackoffInterval(a.assessmentMinInterval, assessment.ConsecutiveFailures))
	assessment.AvailableAtTimestamp = &availableAt

	if err = assessment.Validate(); err != nil {
		a.logger.Errorf("failed to record assessment failure for workspace %q: %v", workspace.FullPath, err)
		return
	}
	if _, err = a.dbClient.WorkspaceAssessments.UpdateWorkspaceAssessment(ctx, assessment); err != nil {
		if errors.ErrorCode(err) == errors.EOptimisticLock {
			// Another writer already updated this row since it was read above — the same
			// benign race the create branch tolerates, not a fault.
			a.logger.Infof("assessment scheduler lost an optimistic-lock race recording a failure for workspace %q: %v", workspace.FullPath, err)
		} else {
			a.logger.Errorf("failed to record assessment failure for workspace %q: %v", workspace.FullPath, err)
		}
	}
}

// startWorkspaceAssessment creates an assessment run for the workspace if one is due. It
// returns true when the in-progress assessment limit has been reached, signaling the caller
// to stop the pass and wait for the next scheduled run. The returned version is whichever
// assessment version (nil if none) was read here, right before the attempt; on failure, the
// caller passes it to handleAssessmentError so recordAssessmentFailure can tell whether the
// row it is about to write to still reflects this attempt, or has moved on since.
func (a *AssessmentScheduler) startWorkspaceAssessment(ctx context.Context, workspace *models.Workspace) (bool, *int, error) {
	assessment, err := a.dbClient.WorkspaceAssessments.GetWorkspaceAssessmentByWorkspaceID(ctx, workspace.Metadata.ID)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get workspace assessment for workspace %q: %w", workspace.FullPath, err)
	}

	var latestAssessmentVersion *int
	if assessment != nil {
		// A recorded failure can push eligibility out past the normal interval (backoff); this
		// mirrors the AND-ed check already applied by the candidate query in GetWorkspaces, so
		// this function stays correct even if called outside that filter (e.g. a future direct
		// call, or a test).
		if assessment.AvailableAtTimestamp != nil && time.Now().Before(*assessment.AvailableAtTimestamp) {
			return false, nil, nil
		}
		// Check if the assessment satisfies the minimum interval since it may have been
		// started manually or by another instance. A stale in-progress assessment (its run
		// was abandoned before completing) is restarted even within the minimum interval.
		if !assessment.IsStaleInProgress() && time.Since(assessment.StartedAtTimestamp) < a.assessmentMinInterval {
			return false, nil, nil
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
		return false, latestAssessmentVersion, err
	}
	if inProgress >= a.assessmentRunLimit {
		// Limit reached: stop the pass and wait for the next scheduled run.
		return true, latestAssessmentVersion, nil
	}

	// Create the assessment run by invoking the command directly; the command processor
	// owns its own transaction. Assessments are system-initiated, so the run's creator is
	// the system actor and no caller is required on the context.
	cmd := a.cmdFactory.NewCreateAssessmentRun(&commands.CreateAssessmentRunInput{
		CreatedBy:               models.SystemActor(),
		WorkspaceID:             workspace.Metadata.ID,
		LatestAssessmentVersion: latestAssessmentVersion,
		// Scheduler-triggered assessments run automatically and frequently, so suppress
		// the run-creation activity event to avoid flooding the activity feed.
		SkipActivityEvent: true,
	})
	if err = a.cmdProcessor.ProcessCommand(ctx, cmd); err != nil {
		// Errors are classified by the caller, which is the single place that decides whether a
		// condition is a benign race, an expected data condition worth recording, or a fault.
		return false, latestAssessmentVersion, err
	}

	assessmentSchedulerAssessmentCount.Inc()

	a.logger.Infof("assessment scheduler created assessment run for workspace %q", workspace.FullPath)

	return false, nil, nil
}
