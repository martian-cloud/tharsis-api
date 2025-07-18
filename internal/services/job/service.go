// Package job package
package job

//go:generate go tool mockery --name Service --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/smithy-go/ptr"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logstream"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run/state"
	rnr "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/runner"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"go.opentelemetry.io/otel/attribute"
)

const (
	// Number of concurrent jobs a given runner can execute.
	runnerJobsLimit int = 100
)

// RunnerAvailabilityStatusType describes a job's runner availability status
type RunnerAvailabilityStatusType string

const (
	// RunnerAvailabilityStatusNoneType indicates no runners are available
	RunnerAvailabilityStatusNoneType RunnerAvailabilityStatusType = "NONE"
	// RunnerAvailabilityStatusInactiveType indicates no active runners are available (but one or more stale ones are)
	RunnerAvailabilityStatusInactiveType RunnerAvailabilityStatusType = "INACTIVE"
	// RunnerAvailabilityStatusAvailableType indicates one or more active runners are available
	RunnerAvailabilityStatusAvailableType RunnerAvailabilityStatusType = "AVAILABLE"
	// RunnerAvailabilityStatusAssignedType indicates the job has been assigned to a runner
	RunnerAvailabilityStatusAssignedType RunnerAvailabilityStatusType = "ASSIGNED"
)

// GetJobsInput includes options for getting jobs
type GetJobsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.JobSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Status is the job status to filter on
	Status *models.JobStatus
	// Type is the job type to filter on
	Type *models.JobType
	// WorkspaceID is the workspace ID to filter on
	WorkspaceID *string
	// RunnerID filters the jobs by the specified runner ID
	RunnerID *string
}

// ClaimJobResponse is returned when a runner claims a Job
type ClaimJobResponse struct {
	JobID string
	Token string
}

// LogStreamEventSubscriptionOptions includes options for setting up a log event subscription
type LogStreamEventSubscriptionOptions struct {
	LastSeenLogSize *int
	JobID           string
}

// LogEvent represents a run event
type LogEvent struct {
	Action string
	JobID  string
	Size   int
}

// CancellationEvent represents a job cancellation event
type CancellationEvent struct {
	Job models.Job
}

// CancellationSubscriptionsOptions includes options for setting up a cancellation event subscription
type CancellationSubscriptionsOptions struct {
	JobID string
}

// SubscribeToJobsInput is the input for subscribing to jobs
type SubscribeToJobsInput struct {
	WorkspaceID *string
	RunnerID    *string
}

// Event is a job event
type Event struct {
	Job    *models.Job
	Action string
}

// Service implements all job related functionality
type Service interface {
	ClaimJob(ctx context.Context, runnerID string) (*ClaimJobResponse, error)
	GetJobByID(ctx context.Context, jobID string) (*models.Job, error)
	GetJobByTRN(ctx context.Context, trn string) (*models.Job, error)
	GetJobsByIDs(ctx context.Context, idList []string) ([]models.Job, error)
	GetJobs(ctx context.Context, input *GetJobsInput) (*db.JobsResult, error)
	GetLatestJobForRun(ctx context.Context, run *models.Run) (*models.Job, error)
	SubscribeToCancellationEvent(ctx context.Context, options *CancellationSubscriptionsOptions) (<-chan *CancellationEvent, error)
	WriteLogs(ctx context.Context, jobID string, startOffset int, logs []byte) (int, error)
	ReadLogs(ctx context.Context, jobID string, startOffset int, limit int) ([]byte, error)
	SubscribeToLogStreamEvents(ctx context.Context, options *LogStreamEventSubscriptionOptions) (<-chan *logstream.LogEvent, error)
	GetLogStreamsByJobIDs(ctx context.Context, idList []string) ([]models.LogStream, error)
	SubscribeToJobs(ctx context.Context, options *SubscribeToJobsInput) (<-chan *Event, error)
	GetRunnerAvailabilityForJob(ctx context.Context, jobID string) (*RunnerAvailabilityStatusType, error)
}

type service struct {
	logger           logger.Logger
	dbClient         *db.Client
	idp              auth.IdentityProvider
	logStreamManager logstream.Manager
	eventManager     *events.EventManager
	runStateManager  state.RunStateManager
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	idp auth.IdentityProvider,
	logStreamManager logstream.Manager,
	eventManager *events.EventManager,
	runStateManager state.RunStateManager,
) Service {
	return &service{logger, dbClient, idp, logStreamManager, eventManager, runStateManager}
}

func (s *service) GetJobByID(ctx context.Context, jobID string) (*models.Job, error) {
	ctx, span := tracer.Start(ctx, "svc.GetJobByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	job, err := s.dbClient.Jobs.GetJobByID(ctx, jobID)
	if err != nil {
		tracing.RecordError(span, err, "Failed to get job")
		return nil, errors.Wrap(
			err,
			"Failed to get job",
		)
	}

	if job == nil {
		tracing.RecordError(span, nil, "Job with ID %s not found", jobID)
		return nil, errors.New("Job with ID %s not found", jobID, errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequirePermission(ctx, models.ViewJobPermission, auth.WithJobID(jobID), auth.WithWorkspaceID(job.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return job, nil
}

func (s *service) GetJobByTRN(ctx context.Context, trn string) (*models.Job, error) {
	ctx, span := tracer.Start(ctx, "svc.GetJobByTRN")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	job, err := s.dbClient.Jobs.GetJobByTRN(ctx, trn)
	if err != nil {
		tracing.RecordError(span, err, "Failed to get job")
		return nil, errors.Wrap(
			err,
			"Failed to get job",
		)
	}

	if job == nil {
		tracing.RecordError(span, nil, "Job with TRN %s not found", trn)
		return nil, errors.New("Job with TRN %s not found", trn, errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequirePermission(ctx, models.ViewJobPermission, auth.WithJobID(job.Metadata.ID), auth.WithWorkspaceID(job.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return job, nil
}

func (s *service) GetJobsByIDs(ctx context.Context, idList []string) ([]models.Job, error) {
	ctx, span := tracer.Start(ctx, "svc.GetJobsByIDs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	resp, err := s.dbClient.Jobs.GetJobs(ctx, &db.GetJobsInput{Filter: &db.JobFilter{JobIDs: idList}})
	if err != nil {
		tracing.RecordError(span, err, "failed to get jobs")
		return nil, err
	}

	// Verify user has access to all returned jobs
	for _, job := range resp.Jobs {
		err = caller.RequirePermission(ctx, models.ViewJobPermission, auth.WithJobID(job.Metadata.ID), auth.WithWorkspaceID(job.WorkspaceID))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
	}

	return resp.Jobs, nil
}

func (s *service) GetJobs(ctx context.Context, input *GetJobsInput) (*db.JobsResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetJobs")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	if input.WorkspaceID != nil {
		rErr := caller.RequirePermission(ctx, models.ViewWorkspacePermission, auth.WithWorkspaceID(*input.WorkspaceID))
		if rErr != nil {
			return nil, rErr
		}
	} else if input.RunnerID != nil {
		runner, rErr := s.dbClient.Runners.GetRunnerByID(ctx, *input.RunnerID)
		if rErr != nil {
			return nil, rErr
		}
		if runner == nil {
			return nil, errors.New("runner not found with ID: %s", *input.RunnerID, errors.WithErrorCode(errors.ENotFound))
		}
		if rErr = rnr.RequireViewerAccessToRunnerResource(ctx, runner); rErr != nil {
			return nil, rErr
		}
	} else if !caller.IsAdmin() {
		return nil, errors.New(
			"Only system admins can subscribe to all job events without filters",
			errors.WithErrorCode(errors.EForbidden),
		)
	}

	dbInput := &db.GetJobsInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.JobFilter{
			JobStatus:   input.Status,
			JobType:     input.Type,
			WorkspaceID: input.WorkspaceID,
			RunnerID:    input.RunnerID,
		},
	}

	jobsResult, err := s.dbClient.Jobs.GetJobs(ctx, dbInput)
	if err != nil {
		tracing.RecordError(span, err, "failed to get jobs")
		return nil, err
	}

	return jobsResult, nil
}

func (s *service) GetLatestJobForRun(ctx context.Context, run *models.Run) (*models.Job, error) {
	ctx, span := tracer.Start(ctx, "svc.GetLatestJobForRun")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	sortBy := db.JobSortableFieldUpdatedAtDesc
	jobsResult, err := s.dbClient.Jobs.GetJobs(ctx, &db.GetJobsInput{
		Sort: &sortBy,
		Filter: &db.JobFilter{
			RunID: &run.Metadata.ID,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get jobs")
		return nil, err
	}

	if len(jobsResult.Jobs) == 0 {
		return nil, nil
	}

	err = caller.RequirePermission(ctx, models.ViewJobPermission, auth.WithJobID(jobsResult.Jobs[0].Metadata.ID), auth.WithWorkspaceID(run.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return &jobsResult.Jobs[0], nil
}

func (s *service) SubscribeToJobs(ctx context.Context, options *SubscribeToJobsInput) (<-chan *Event, error) {
	ctx, span := tracer.Start(ctx, "svc.SubscribeToJobs")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if options.WorkspaceID != nil {
		err := caller.RequirePermission(ctx, models.ViewWorkspacePermission, auth.WithWorkspaceID(*options.WorkspaceID))
		if err != nil {
			return nil, err
		}
	} else if options.RunnerID != nil {
		runner, rErr := s.dbClient.Runners.GetRunnerByID(ctx, *options.RunnerID)
		if rErr != nil {
			return nil, rErr
		}
		if runner == nil {
			return nil, errors.New("runner not found with ID: %s", *options.RunnerID, errors.WithErrorCode(errors.ENotFound))
		}
		if rErr = rnr.RequireViewerAccessToRunnerResource(ctx, runner); rErr != nil {
			return nil, rErr
		}
	} else if !caller.IsAdmin() {
		return nil, errors.New(
			"Only system admins can subscribe to all job events without filters",
			errors.WithErrorCode(errors.EForbidden),
		)
	}

	subscription := events.Subscription{
		Type: events.JobSubscription,
		Actions: []events.SubscriptionAction{
			events.CreateAction,
			events.UpdateAction,
		},
	}

	subscriber := s.eventManager.Subscribe([]events.Subscription{subscription})

	outgoing := make(chan *Event)
	go func() {
		// Defer close of outgoing channel
		defer close(outgoing)
		defer s.eventManager.Unsubscribe(subscriber)

		// Wait for job updates
		for {
			event, err := subscriber.GetEvent(ctx)
			if err != nil {
				if !errors.IsContextCanceledError(err) {
					s.logger.Errorf("error occurred while waiting for job events: %v", err)
				}
				return
			}

			eventData, err := event.ToJobEventData()
			if err != nil {
				s.logger.Errorf("failed to get job event data in job subscription: %v", err)
				continue
			}

			// Check if this event is for the runner we're interested in
			if options.RunnerID != nil && (eventData.RunnerID == nil || *eventData.RunnerID != *options.RunnerID) {
				continue
			}

			// Check if this event is for the workspace we're interested in
			if options.WorkspaceID != nil && eventData.WorkspaceID != *options.WorkspaceID {
				continue
			}

			job, err := s.dbClient.Jobs.GetJobByID(ctx, event.ID)
			if err != nil {
				s.logger.Errorf("error querying for job in subscription goroutine: %v", err)
				continue
			}
			if job == nil {
				s.logger.Errorf("received event for job that does not exist %s", event.ID)
				continue
			}

			select {
			case <-ctx.Done():
				return
			case outgoing <- &Event{Job: job, Action: event.Action}:
			}
		}
	}()

	return outgoing, nil
}

func (s *service) SubscribeToCancellationEvent(ctx context.Context, options *CancellationSubscriptionsOptions) (<-chan *CancellationEvent, error) {
	outerCtx := ctx // for goroutine
	ctx, span := tracer.Start(ctx, "svc.SubscribeToCancellationEvent")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	jobID := options.JobID

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	job, err := s.GetJobByID(ctx, jobID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get job")
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.ViewJobPermission, auth.WithJobID(job.Metadata.ID), auth.WithWorkspaceID(job.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	subscription := events.Subscription{
		Type:    events.JobSubscription,
		ID:      jobID,
		Actions: []events.SubscriptionAction{events.UpdateAction},
	}
	subscriber := s.eventManager.Subscribe([]events.Subscription{subscription})

	outgoing := make(chan *CancellationEvent)

	go func() {
		defer close(outgoing)
		defer s.eventManager.Unsubscribe(subscriber)

		// Because this goroutine will survive after the parent function returns,
		// this span is not nested inside that for the parent function.
		innerCtx, innerSpan := tracer.Start(outerCtx, "svc.SubscribeToCancellationEvent.goroutine")
		defer innerSpan.End()

		// Query for the job after the subscription is setup to ensure no events are missed
		job, err := s.GetJobByID(innerCtx, jobID)
		if err != nil {
			tracing.RecordError(innerSpan, err, "Error occurred while checking for job cancellation")
			s.logger.Errorf("Error occurred while checking for job cancellation: %v", err)
			return
		}

		if job.CancelRequested {
			select {
			case <-innerCtx.Done():
			case outgoing <- &CancellationEvent{Job: *job}:
			}
			return
		}

		// Wait for cancellation event updates
		for {
			event, err := subscriber.GetEvent(innerCtx)
			if err != nil {
				if !errors.IsContextCanceledError(err) {
					tracing.RecordError(innerSpan, err, "Error occurred while waiting for job cancellation events")
					s.logger.Errorf("Error occurred while waiting for job cancellation events: %v", err)
				}
				return
			}

			eventData, err := event.ToJobEventData()
			if err != nil {
				s.logger.Errorf("failed to get job event data in job event subscription: %v", err)
				continue
			}

			if !eventData.CancelRequested {
				continue
			}

			job, err := s.GetJobByID(innerCtx, event.ID)
			if err != nil {
				if errors.IsContextCanceledError(err) {
					return
				}
				tracing.RecordError(innerSpan, err,
					"Error occurred while querying for job associated with cancellation event %s", event.ID)
				s.logger.Errorf("Error occurred while querying for job associated with cancellation event %s: %v", event.ID, err)
				return
			}

			if job == nil {
				tracing.RecordError(innerSpan, nil, "Job not found for event with ID %s", event.ID)
				s.logger.Errorf("Job not found for event with ID %s", event.ID)
				continue
			}

			select {
			case <-innerCtx.Done():
				return
			case outgoing <- &CancellationEvent{Job: *job}:
			}
		}
	}()

	return outgoing, nil
}

func (s *service) ClaimJob(ctx context.Context, runnerID string) (*ClaimJobResponse, error) {
	ctx, span := tracer.Start(ctx, "svc.ClaimJob")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	runner, err := s.dbClient.Runners.GetRunnerByID(ctx, runnerID)
	if err != nil {
		return nil, err
	}

	if runner == nil {
		return nil, errors.New("runner with id %s not found", errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequirePermission(ctx, models.ClaimJobPermission, auth.WithRunnerID(runner.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	for {
		job, err := s.getNextAvailableQueuedJob(ctx, runner.Metadata.ID)
		if err != nil {
			tracing.RecordError(span, err, "failed to get next available queued job")
			return nil, err
		}

		// Attempt to claim job
		now := time.Now()
		job.Timestamps.PendingTimestamp = &now
		job.Status = models.JobPending
		job.RunnerID = &runner.Metadata.ID
		job.RunnerPath = ptr.String(runner.GetResourcePath())

		job, err = s.runStateManager.UpdateJob(ctx, job)
		if err != nil {
			if err == db.ErrOptimisticLockError {
				continue
			}
			tracing.RecordError(span, err, "failed to update job")
			return nil, err
		}

		if job != nil {
			maxJobDuration := time.Duration(job.MaxJobDuration) * time.Minute
			expiration := time.Now().Add(maxJobDuration + time.Hour)
			token, err := s.idp.GenerateToken(ctx, &auth.TokenInput{
				// Expiration is job timeout plus 1 hour to give the job time to gracefully exit
				Expiration: &expiration,
				Subject:    fmt.Sprintf("job-%s", job.Metadata.ID),
				Claims: map[string]string{
					"job_id":       job.GetGlobalID(),
					"run_id":       gid.ToGlobalID(types.RunModelType, job.RunID),
					"workspace_id": gid.ToGlobalID(types.WorkspaceModelType, job.WorkspaceID),
					"type":         auth.JobTokenType,
				},
			})
			if err != nil {
				tracing.RecordError(span, err, "failed to generate token")
				return nil, err
			}

			s.logger.Infow("Claimed a job.",
				"caller", caller.GetSubject(),
				"workspaceID", job.WorkspaceID,
				"jobID", job.Metadata.ID,
			)
			return &ClaimJobResponse{JobID: job.Metadata.ID, Token: string(token)}, nil
		}
	}
}

func (s *service) SubscribeToLogStreamEvents(ctx context.Context, options *LogStreamEventSubscriptionOptions) (<-chan *logstream.LogEvent, error) {
	ctx, span := tracer.Start(ctx, "svc.SubscribeToLogStreamEvents")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	job, err := s.getJobByID(ctx, options.JobID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get job by ID")
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.ViewJobPermission, auth.WithJobID(job.Metadata.ID),
		auth.WithWorkspaceID(job.WorkspaceID))
	if err != nil {
		return nil, err
	}

	logStream, err := s.dbClient.LogStreams.GetLogStreamByJobID(ctx, job.Metadata.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get log stream by job ID", errors.WithSpan(span))
	}

	if logStream == nil {
		return nil, fmt.Errorf("log stream not found for job %s", job.Metadata.ID)
	}

	return s.logStreamManager.Subscribe(ctx, &logstream.SubscriptionOptions{
		LastSeenLogSize: options.LastSeenLogSize,
		LogStreamID:     logStream.Metadata.ID,
	})
}

func (s *service) WriteLogs(ctx context.Context, jobID string, startOffset int, logs []byte) (int, error) {
	ctx, span := tracer.Start(ctx, "svc.WriteLogs")
	span.SetAttributes(attribute.String("job_id", jobID))
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return 0, err
	}

	job, err := s.getJobByID(ctx, jobID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get job by ID")
		return 0, err
	}

	err = caller.RequirePermission(ctx, models.UpdateJobPermission, auth.WithJobID(jobID),
		auth.WithWorkspaceID(job.WorkspaceID))
	if err != nil {
		return 0, err
	}

	stream, err := s.dbClient.LogStreams.GetLogStreamByJobID(ctx, jobID)
	if err != nil {
		return 0, err
	}

	if stream == nil {
		return 0, errors.New("log stream not found for job %s", jobID)
	}

	// Write logs to store
	updatedStream, err := s.logStreamManager.WriteLogs(ctx, stream.Metadata.ID, startOffset, logs)
	if err != nil {
		return 0, err
	}

	return updatedStream.Size, nil
}

func (s *service) ReadLogs(ctx context.Context, jobID string, startOffset int, limit int) ([]byte, error) {
	ctx, span := tracer.Start(ctx, "svc.ReadLogs")
	span.SetAttributes(attribute.String("job_id", jobID))
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	job, err := s.getJobByID(ctx, jobID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get job by ID")
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.ViewJobPermission, auth.WithJobID(jobID),
		auth.WithWorkspaceID(job.WorkspaceID))
	if err != nil {
		return nil, err
	}

	// Find log stream associated with job
	stream, err := s.dbClient.LogStreams.GetLogStreamByJobID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if stream == nil {
		return nil, errors.New("log stream not found %s", jobID)
	}

	return s.logStreamManager.ReadLogs(ctx, stream.Metadata.ID, startOffset, limit)
}

// GetRunnerAvailabilityForJob returns a job's runner status, whether there exists a runner with an active session,
// a runner with a stale session, or no runner that could claim the job.
func (s *service) GetRunnerAvailabilityForJob(ctx context.Context, jobID string) (*RunnerAvailabilityStatusType, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunnerAvailabilityForJob")
	defer span.End()
	runnerAvailability := RunnerAvailabilityStatusInactiveType

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "caller authorization failed", errors.WithSpan(span))
	}

	job, err := s.dbClient.Jobs.GetJobByID(ctx, jobID)
	if err != nil {
		return nil, errors.Wrap(
			err,
			"Failed to get job",
			errors.WithSpan(span),
		)
	}

	if job == nil {
		return nil, errors.New("Job with ID %s not found", jobID,
			errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	err = caller.RequirePermission(ctx, models.ViewJobPermission, auth.WithJobID(jobID), auth.WithWorkspaceID(job.WorkspaceID))
	if err != nil {
		return nil, errors.Wrap(err, "permission check failed", errors.WithSpan(span))
	}

	if job.RunnerID != nil {
		runnerAvailability = RunnerAvailabilityStatusAssignedType
		return &runnerAvailability, nil
	}

	runners, err := s.getRunnersForWorkspace(ctx, job.WorkspaceID, job.Tags)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get runners", errors.WithSpan(span))
	}

	if len(runners) == 0 {
		runnerAvailability = RunnerAvailabilityStatusNoneType
		return &runnerAvailability, nil
	}

	for _, runner := range runners {
		runnerSession, err := s.getRunnerSession(ctx, runner.Metadata.ID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get runner sessions", errors.WithSpan(span))
		}

		if runnerSession != nil {
			if runnerSession.Active() {
				// Any active runner means we should return available.
				runnerAvailability = RunnerAvailabilityStatusAvailableType
				return &runnerAvailability, nil
			}
		}
	}

	// If we get to this point, there was at least one runner but no runner was active.
	return &runnerAvailability, nil
}

func (s *service) getRunnerSession(ctx context.Context, runnerID string) (*models.RunnerSession, error) {
	// Get the most recently contacted session first.
	toSort := db.RunnerSessionSortableFieldLastContactedAtDesc

	sessionsResult, err := s.dbClient.RunnerSessions.GetRunnerSessions(ctx, &db.GetRunnerSessionsInput{
		Sort: &toSort,
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(1),
		},
		Filter: &db.RunnerSessionFilter{
			RunnerID: &runnerID,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get runner sessions")
	}

	var foundSession *models.RunnerSession
	if len(sessionsResult.RunnerSessions) > 0 {
		foundSession = &sessionsResult.RunnerSessions[0]
	}

	return foundSession, nil
}

// getRunnersForWorkspace returns a list of shared and group runners that could possibly claim a job in the workspace
// The caller should wrap any errors with the span.
func (s *service) getRunnersForWorkspace(ctx context.Context,
	workspaceID string, jobTags []string) ([]models.Runner, error) {

	workspace, err := s.dbClient.Workspaces.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace by ID")
	}

	if workspace == nil {
		return nil, errors.New("workspace not found", errors.WithErrorCode(errors.ENotFound))
	}

	tagFilter := &db.RunnerTagFilter{
		TagSubset: jobTags,
	}
	if len(jobTags) == 0 {
		tagFilter.RunUntaggedJobs = ptr.Bool(true)
	}

	sharedRunnerType := models.SharedRunnerType
	sharedRunners, err := s.dbClient.Runners.GetRunners(ctx, &db.GetRunnersInput{
		Filter: &db.RunnerFilter{
			RunnerType: &sharedRunnerType,
			Enabled:    ptr.Bool(true),
			TagFilter:  tagFilter,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get runners")
	}

	groupRunnerType := models.GroupRunnerType
	groupRunners, err := s.dbClient.Runners.GetRunners(ctx, &db.GetRunnersInput{
		Filter: &db.RunnerFilter{
			RunnerType:     &groupRunnerType,
			NamespacePaths: workspace.ExpandPath(),
			Enabled:        ptr.Bool(true),
			TagFilter:      tagFilter,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get runners")
	}

	return append(sharedRunners.Runners, groupRunners.Runners...), nil
}

// getJobByID returns a non-nil job.
func (s *service) getJobByID(ctx context.Context, jobID string) (*models.Job, error) {

	job, err := s.dbClient.Jobs.GetJobByID(ctx, jobID)
	if err != nil {
		return nil, errors.Wrap(
			err,
			"Failed to get job",
			errors.WithErrorCode(errors.EInternal),
		)
	}

	if job == nil {
		return nil, errors.New("Job with ID %s not found", jobID, errors.WithErrorCode(errors.ENotFound))
	}

	return job, nil
}

func (s *service) GetLogStreamsByJobIDs(ctx context.Context, idList []string) ([]models.LogStream, error) {
	ctx, span := tracer.Start(ctx, "svc.GetLogStreamsByJobIDs")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := s.dbClient.Jobs.GetJobs(ctx, &db.GetJobsInput{Filter: &db.JobFilter{JobIDs: idList}})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get jobs", errors.WithSpan(span))
	}

	// Verify user has access to all returned jobs
	for _, job := range resp.Jobs {
		err = caller.RequirePermission(ctx, models.ViewJobPermission, auth.WithJobID(job.Metadata.ID),
			auth.WithWorkspaceID(job.WorkspaceID))
		if err != nil {
			return nil, err
		}
	}

	if len(resp.Jobs) > 0 {
		result, err := s.dbClient.LogStreams.GetLogStreams(ctx, &db.GetLogStreamsInput{
			Filter: &db.LogStreamFilter{
				JobIDs: idList,
			},
		})
		if err != nil {
			return nil, err
		}

		return result.LogStreams, nil
	}

	return []models.LogStream{}, nil
}

func (s *service) getNextAvailableQueuedJob(ctx context.Context, runnerID string) (*models.Job, error) {
	// Subscribe to job create and update events
	jobSubscription := events.Subscription{
		Type:    events.JobSubscription,
		Actions: []events.SubscriptionAction{events.CreateAction, events.UpdateAction},
	}
	// Subscribe to runner events because a runner may become available
	runnerSubscription := events.Subscription{
		Type:    events.RunnerSubscription,
		Actions: []events.SubscriptionAction{},
	}
	// Subscribe to workspace delete events because deleting a workspace may cause a job in a different workspace to become available
	workspaceSubscription := events.Subscription{
		Type:    events.WorkspaceSubscription,
		Actions: []events.SubscriptionAction{events.UpdateAction, events.DeleteAction},
	}

	// Subscribe to job and run events
	subscriber := s.eventManager.Subscribe([]events.Subscription{jobSubscription, runnerSubscription, workspaceSubscription})
	defer s.eventManager.Unsubscribe(subscriber)

	// Wait for next available run
	for {
		job, err := s.getNextAvailableJob(ctx, runnerID)
		if err != nil {
			return nil, err
		}

		if job != nil {
			return job, nil
		}

		_, err = subscriber.GetEvent(ctx)
		if err != nil {
			return nil, err
		}
	}
}

// isRunnerBelowJobsLimit determines if runner is full.
func (s *service) isRunnerBelowJobsLimit(ctx context.Context, runner *models.Runner) (bool, error) {
	runnerJobsCount, err := s.dbClient.Jobs.GetJobCountForRunner(ctx, runner.Metadata.ID)
	if err != nil {
		return false, err
	}
	return runnerJobsCount < runnerJobsLimit, nil
}

// getNextAvailableJob returns a new job when workspace doesn't have an active job
// and the runner is not full.
func (s *service) getNextAvailableJob(ctx context.Context, runnerID string) (*models.Job, error) {
	runner, err := s.dbClient.Runners.GetRunnerByID(ctx, runnerID)
	if err != nil {
		return nil, err
	}

	if runner == nil {
		return nil, errors.New("runner was deleted", errors.WithErrorCode(errors.ENotFound))
	}

	if runner.Disabled {
		// Return nil since runner is disabled
		return nil, nil
	}

	runnerTags := runner.Tags
	if runnerTags == nil {
		runnerTags = []string{}
	}

	tagFilter := &db.JobTagFilter{
		TagSuperset: runnerTags,
	}

	if !runner.RunUntaggedJobs {
		tagFilter.ExcludeUntaggedJobs = ptr.Bool(true)
	}

	// Request next available Job
	queuedStatus := models.JobQueued
	sortBy := db.JobSortableFieldCreatedAtAsc
	jobsResult, err := s.dbClient.Jobs.GetJobs(ctx, &db.GetJobsInput{
		Sort: &sortBy,
		Filter: &db.JobFilter{
			JobStatus: &queuedStatus,
			TagFilter: tagFilter,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to request next job %v", err)
	}

	for _, job := range jobsResult.Jobs {
		ws, err := s.dbClient.Workspaces.GetWorkspaceByID(ctx, job.WorkspaceID)
		if err != nil {
			return nil, err
		}

		if ws == nil {
			// This will only occur if the workspace is deleted after the job is queried
			continue
		}

		if ws.Locked {
			continue
		}

		if runner.Type == models.GroupRunnerType {
			// Check if this group runner is in an ancestor group of the job's workspace to claim this job
			runnerGroupPath := runner.GetGroupPath()
			if runnerGroupPath != ws.GetGroupPath() {
				if !strings.HasPrefix(ws.GetGroupPath(), fmt.Sprintf("%s/", runnerGroupPath)) {
					continue
				}
			}
		}

		below, err := s.isRunnerBelowJobsLimit(ctx, runner)
		if err != nil {
			return nil, err
		}
		if below {
			return &job, nil
		}
	}
	return nil, nil
}
