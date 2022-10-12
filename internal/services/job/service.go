package job

//go:generate mockery --name Service --inpackage --case underscore

import (
	"context"
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

const (

	// Number of concurrent jobs a given runner can execute.
	runnerJobsLimit int = 100
)

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

// LogEventSubscriptionOptions includes options for setting up a log event subscription
type LogEventSubscriptionOptions struct {
	LastSeenLogSize *int
}

// CancellationSubscriptionsOptions includes options for setting up a cancellation event subscription
type CancellationSubscriptionsOptions struct {
	JobID string
}

// ClaimJobResponse is returned when a runner claims a Job
type ClaimJobResponse struct {
	Job   *models.Job
	Token string
}

// Service implements all job related functionality
type Service interface {
	GetNextAvailableQueuedJob(ctx context.Context, runnerID string) (*models.Job, error)
	GetJob(ctx context.Context, jobID string) (*models.Job, error)
	GetJobsByIDs(ctx context.Context, idList []string) ([]models.Job, error)
	GetLatestJobForRun(ctx context.Context, run *models.Run) (*models.Job, error)
	SubscribeToCancellationEvent(ctx context.Context, options *CancellationSubscriptionsOptions) (<-chan *CancellationEvent, error)
	SaveLogs(ctx context.Context, jobID string, startOffset int, buffer []byte) error
	GetLogs(ctx context.Context, jobID string, startOffset int, limit int) ([]byte, error)
	GetJobLogDescriptor(ctx context.Context, job *models.Job) (*models.JobLogDescriptor, error)
	SubscribeToJobLogEvents(ctx context.Context, job *models.Job, options *LogEventSubscriptionOptions) (<-chan *LogEvent, error)
}

type service struct {
	logger       logger.Logger
	dbClient     *db.Client
	eventManager *events.EventManager
	logStore     LogStore
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	eventManager *events.EventManager,
	logStore LogStore,
) Service {
	return &service{logger, dbClient, eventManager, logStore}
}

func (s *service) SubscribeToJobLogEvents(ctx context.Context, job *models.Job, options *LogEventSubscriptionOptions) (<-chan *LogEvent, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err := caller.RequireAccessToWorkspace(ctx, job.WorkspaceID, models.ViewerRole); err != nil {
		return nil, err
	}

	outgoing := make(chan *LogEvent)

	go func() {
		// Defer close of outgoing channel
		defer close(outgoing)

		subscription := events.Subscription{
			Type: events.JobLogSubscription,
			Actions: []events.SubscriptionAction{
				events.CreateAction,
				events.UpdateAction,
			},
		}
		subscriber := s.eventManager.Subscribe([]events.Subscription{subscription})

		defer s.eventManager.Unsubscribe(subscriber)

		if options.LastSeenLogSize != nil {
			descriptor, err := s.dbClient.Jobs.GetJobLogDescriptorByJobID(ctx, job.Metadata.ID)
			if err != nil {
				return
			}

			var size int
			if descriptor != nil {
				size = descriptor.Size
			}

			if size != *options.LastSeenLogSize {
				select {
				case <-ctx.Done():
					return
				case outgoing <- &LogEvent{Action: string(events.UpdateAction), JobID: job.Metadata.ID, Size: size}:
				}
			}
		}

		// Wait for job updates
		for {
			event, err := subscriber.GetEvent(ctx)
			if err != nil {
				if err != context.Canceled {
					s.logger.Errorf("Error occurred while waiting for job log events: %v", err)
				}
				return
			}

			descriptor, err := s.dbClient.Jobs.GetJobLogDescriptor(ctx, event.ID)
			if err != nil {
				s.logger.Errorf("Error occurred while querying for job log descriptor associated with job log event %s: %v", event.ID, err)
				return
			}

			if descriptor == nil {
				s.logger.Errorf("Error occurred while querying for job log descriptor associated with job log event %s: descriptor not found", event.ID)
				continue
			}

			// Only return events for job log descriptors that match the job ID
			if descriptor.JobID != job.Metadata.ID {
				continue
			}

			select {
			case <-ctx.Done():
				return
			case outgoing <- &LogEvent{Action: event.Action, JobID: descriptor.JobID, Size: descriptor.Size}:
			}
		}
	}()

	return outgoing, nil
}

func (s *service) GetJobLogDescriptor(ctx context.Context, job *models.Job) (*models.JobLogDescriptor, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err = caller.RequireAccessToWorkspace(ctx, job.WorkspaceID, models.ViewerRole); err != nil {
		return nil, err
	}

	descriptor, err := s.dbClient.Jobs.GetJobLogDescriptorByJobID(ctx, job.Metadata.ID)
	if err != nil {
		return nil, err
	}

	return descriptor, nil
}

func (s *service) GetJob(ctx context.Context, jobID string) (*models.Job, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	job, err := s.dbClient.Jobs.GetJobByID(ctx, jobID)
	if err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to get job",
			errors.WithErrorErr(err),
		)
	}

	if job == nil {
		return nil, errors.NewError(
			errors.ENotFound,
			fmt.Sprintf("Job with ID %s not found", jobID),
		)
	}

	if err := caller.RequireAccessToWorkspace(ctx, job.WorkspaceID, models.ViewerRole); err != nil {
		return nil, err
	}

	return job, nil
}

func (s *service) GetJobsByIDs(ctx context.Context, idList []string) ([]models.Job, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := s.dbClient.Jobs.GetJobs(ctx, &db.GetJobsInput{Filter: &db.JobFilter{JobIDs: idList}})
	if err != nil {
		return nil, err
	}

	// Verify user has access to all returned jobs
	for _, job := range resp.Jobs {
		if err := caller.RequireAccessToWorkspace(ctx, job.WorkspaceID, models.ViewerRole); err != nil {
			return nil, err
		}
	}

	return resp.Jobs, nil
}

func (s *service) GetLatestJobForRun(ctx context.Context, run *models.Run) (*models.Job, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err = caller.RequireAccessToWorkspace(ctx, run.WorkspaceID, models.ViewerRole); err != nil {
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
		return nil, err
	}

	if len(jobsResult.Jobs) == 0 {
		return nil, nil
	}

	return &jobsResult.Jobs[0], nil
}

func (s *service) SubscribeToCancellationEvent(ctx context.Context, options *CancellationSubscriptionsOptions) (<-chan *CancellationEvent, error) {
	jobID := options.JobID

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	job, err := s.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}

	if err := caller.RequireAccessToWorkspace(ctx, job.WorkspaceID, models.ViewerRole); err != nil {
		return nil, err
	}

	outgoing := make(chan *CancellationEvent)

	go func() {
		defer close(outgoing)

		subscription := events.Subscription{
			Type:    events.JobSubscription,
			ID:      jobID,
			Actions: []events.SubscriptionAction{events.UpdateAction},
		}
		subscriber := s.eventManager.Subscribe([]events.Subscription{subscription})
		defer s.eventManager.Unsubscribe(subscriber)

		// Query for the job after the subscription is setup to ensure no events are missed
		job, err := s.GetJob(ctx, jobID)
		if err != nil {
			s.logger.Errorf("Error occurred while checking for job cancellation: %v", err)
			return
		}

		if job.CancelRequested {
			outgoing <- &CancellationEvent{Job: *job}
			return
		}

		// Wait for job updates
		for {
			event, err := subscriber.GetEvent(ctx)
			if err != nil {
				if err != context.Canceled {
					s.logger.Errorf("Error occurred while waiting for job cancellation events: %v", err)
				}
				return
			}

			job, err := s.GetJob(ctx, event.ID)
			if err != nil {
				s.logger.Errorf("Error occurred while querying for job associated with cancellation event %s: %v", event.ID, err)
				return
			}

			if job == nil {
				s.logger.Errorf("Job not found for event with ID %s", event.ID)
				continue
			}

			if !job.CancelRequested {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case outgoing <- &CancellationEvent{Job: *job}:
			}
		}
	}()

	return outgoing, nil
}

func (s *service) SaveLogs(ctx context.Context, jobID string, startOffset int, buffer []byte) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	if err = caller.RequireJobWriteAccess(ctx, jobID); err != nil {
		return err
	}

	job, err := s.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	if err := s.logStore.SaveLogs(ctx, job.WorkspaceID, job.RunID, jobID, startOffset, buffer); err != nil {
		return errors.NewError(errors.EInvalid, "Failed to save logs", errors.WithErrorErr(err))
	}

	return nil
}

func (s *service) GetLogs(ctx context.Context, jobID string, startOffset int, limit int) ([]byte, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	job, err := s.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}

	if err := caller.RequireAccessToWorkspace(ctx, job.WorkspaceID, models.ViewerRole); err != nil {
		return nil, err
	}

	if limit < 0 || startOffset < 0 {
		return nil, errors.NewError(errors.EInvalid, "limit and offset cannot be negative")
	}

	return s.logStore.GetLogs(ctx, job.WorkspaceID, job.RunID, jobID, startOffset, limit)
}

func (s *service) GetNextAvailableQueuedJob(ctx context.Context, runnerID string) (*models.Job, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// Only allow system caller for now until runner registration is supported
	if _, ok := caller.(*auth.SystemCaller); !ok {
		return nil, errors.NewError(errors.EForbidden, fmt.Sprintf("Subject %s is not authorized to get queued jobs", caller.GetSubject()))
	}

	subscription := events.Subscription{
		Type:    events.JobSubscription,
		Actions: []events.SubscriptionAction{},
	}
	subscriber := s.eventManager.Subscribe([]events.Subscription{subscription})
	defer s.eventManager.Unsubscribe(subscriber)

	job, err := s.getNextAvailableJob(ctx, runnerID)
	if err != nil {
		return nil, err
	}

	if job != nil {
		return job, nil
	}

	// Wait for next available run
	for {
		event, err := subscriber.GetEvent(ctx)
		if err != nil {
			return nil, err
		}

		s.logger.Info("Received job notification from db")

		if events.SubscriptionAction(event.Action) == events.DeleteAction {
			nextJob, err := s.getNextAvailableJob(ctx, runnerID)
			if err != nil {
				return nil, err
			}
			if nextJob != nil {
				return nextJob, nil
			}
		} else {
			job, err := s.dbClient.Jobs.GetJobByID(ctx, event.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to request job %v", err)
			}

			if job == nil {
				s.logger.Errorf("Job not found for event with ID %s", event.ID)
				continue
			}

			// Return the queued job if current job has finished.
			if job.Status == models.JobQueued {
				ws, err := s.dbClient.Workspaces.GetWorkspaceByID(ctx, job.WorkspaceID)
				if err != nil {
					return nil, err
				}

				if !ws.Locked {
					below, err := s.isRunnerBelowJobsLimit(ctx, runnerID)
					if err != nil {
						return nil, err
					}

					if below {
						return job, nil
					}
				}
			}

			// Find any queued jobs once current job has finished.
			if job.Status == models.JobFinished {
				nextJob, err := s.getNextAvailableJob(ctx, runnerID)
				if err != nil {
					return nil, err
				}
				if nextJob != nil {
					return nextJob, nil
				}
			}
		}
	}
}

// isRunnerBelowJobsLimit determines if runner is full.
func (s *service) isRunnerBelowJobsLimit(ctx context.Context, runnerID string) (bool, error) {
	runnerJobsCount, err := s.dbClient.Jobs.GetJobCountForRunner(ctx, runnerID)
	if err != nil {
		return false, err
	}
	return runnerJobsCount < runnerJobsLimit, nil
}

// getNextAvailableJob returns a new job when workspace doesn't have an active job
// and the runner is not full.
func (s *service) getNextAvailableJob(ctx context.Context, runnerID string) (*models.Job, error) {
	// Request next available Job
	queuedStatus := models.JobQueued
	sortBy := db.JobSortableFieldCreatedAtAsc
	jobsResult, err := s.dbClient.Jobs.GetJobs(ctx, &db.GetJobsInput{
		Sort: &sortBy,
		Filter: &db.JobFilter{
			JobStatus: &queuedStatus,
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

		if !ws.Locked {
			below, err := s.isRunnerBelowJobsLimit(ctx, runnerID)
			if err != nil {
				return nil, err
			}
			if below {
				return &job, nil
			}
		}
	}
	return nil, nil
}
