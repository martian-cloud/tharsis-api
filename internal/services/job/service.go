package job

//go:generate mockery --name Service --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/smithy-go/ptr"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run/state"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const (
	// Number of concurrent jobs a given runner can execute.
	runnerJobsLimit int = 100
)

// ClaimJobResponse is returned when a runner claims a Job
type ClaimJobResponse struct {
	JobID string
	Token string
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

// LogEventSubscriptionOptions includes options for setting up a log event subscription
type LogEventSubscriptionOptions struct {
	LastSeenLogSize *int
}

// CancellationSubscriptionsOptions includes options for setting up a cancellation event subscription
type CancellationSubscriptionsOptions struct {
	JobID string
}

// Service implements all job related functionality
type Service interface {
	ClaimJob(ctx context.Context, runnerPath string) (*ClaimJobResponse, error)
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
	logger          logger.Logger
	dbClient        *db.Client
	idp             *auth.IdentityProvider
	eventManager    *events.EventManager
	runStateManager *state.RunStateManager
	logStore        LogStore
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	idp *auth.IdentityProvider,
	eventManager *events.EventManager,
	runStateManager *state.RunStateManager,
	logStore LogStore,
) Service {
	return &service{logger, dbClient, idp, eventManager, runStateManager, logStore}
}

func (s *service) SubscribeToJobLogEvents(ctx context.Context, job *models.Job, options *LogEventSubscriptionOptions) (<-chan *LogEvent, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.ViewJobPermission, auth.WithJobID(job.Metadata.ID), auth.WithWorkspaceID(job.WorkspaceID))
	if err != nil {
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

	err = caller.RequirePermission(ctx, permissions.ViewJobPermission, auth.WithJobID(job.Metadata.ID), auth.WithWorkspaceID(job.WorkspaceID))
	if err != nil {
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
		return nil, errors.Wrap(
			err,
			errors.EInternal,
			"Failed to get job",
		)
	}

	if job == nil {
		return nil, errors.New(errors.ENotFound, "Job with ID %s not found", jobID)
	}

	err = caller.RequirePermission(ctx, permissions.ViewJobPermission, auth.WithJobID(jobID), auth.WithWorkspaceID(job.WorkspaceID))
	if err != nil {
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
		err = caller.RequirePermission(ctx, permissions.ViewJobPermission, auth.WithJobID(job.Metadata.ID), auth.WithWorkspaceID(job.WorkspaceID))
		if err != nil {
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

	err = caller.RequirePermission(ctx, permissions.ViewJobPermission, auth.WithJobID(jobsResult.Jobs[0].Metadata.ID), auth.WithWorkspaceID(run.WorkspaceID))
	if err != nil {
		return nil, err
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

	err = caller.RequirePermission(ctx, permissions.ViewJobPermission, auth.WithJobID(job.Metadata.ID), auth.WithWorkspaceID(job.WorkspaceID))
	if err != nil {
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

	job, err := s.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateJobPermission, auth.WithJobID(jobID), auth.WithWorkspaceID(job.WorkspaceID))
	if err != nil {
		return err
	}

	if err := s.logStore.SaveLogs(ctx, job.WorkspaceID, job.RunID, jobID, startOffset, buffer); err != nil {
		return errors.Wrap(err, errors.EInvalid, "Failed to save logs")
	}

	return nil
}

func (s *service) GetLogs(ctx context.Context, jobID string, startOffset int, limit int) ([]byte, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if limit < 0 || startOffset < 0 {
		return nil, errors.New(errors.EInvalid, "limit and offset cannot be negative")
	}

	job, err := s.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.ViewJobPermission, auth.WithJobID(jobID), auth.WithWorkspaceID(job.WorkspaceID))
	if err != nil {
		return nil, err
	}

	return s.logStore.GetLogs(ctx, job.WorkspaceID, job.RunID, jobID, startOffset, limit)
}

func (s *service) ClaimJob(ctx context.Context, runnerPath string) (*ClaimJobResponse, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// Find runner by path
	pathParts := strings.Split(runnerPath, "/")
	getRunnerInput := db.GetRunnersInput{
		Filter: &db.RunnerFilter{
			RunnerName: ptr.String(pathParts[len(pathParts)-1]),
		},
	}

	if len(pathParts) > 1 {
		groupPath := strings.Join(pathParts[:len(pathParts)-1], "/")
		group, ggErr := s.dbClient.Groups.GetGroupByFullPath(ctx, groupPath)
		if ggErr != nil {
			return nil, ggErr
		}
		if group == nil {
			return nil, errors.New(errors.ENotFound, "runner not found")
		}
		getRunnerInput.Filter.GroupID = &group.Metadata.ID
	}

	runnersResp, err := s.dbClient.Runners.GetRunners(ctx, &getRunnerInput)
	if err != nil {
		return nil, err
	}

	if len(runnersResp.Runners) == 0 {
		return nil, errors.New(errors.ENotFound, "runner not found")
	}

	runner := runnersResp.Runners[0]

	err = caller.RequirePermission(ctx, permissions.ClaimJobPermission, auth.WithRunnerID(runner.Metadata.ID))
	if err != nil {
		return nil, err
	}

	for {
		job, err := s.getNextAvailableQueuedJob(ctx, &runner)
		if err != nil {
			return nil, err
		}

		// Attempt to claim job
		now := time.Now()
		job.Timestamps.PendingTimestamp = &now
		job.Status = models.JobPending
		job.RunnerID = &runner.Metadata.ID
		job.RunnerPath = &runner.ResourcePath

		job, err = s.runStateManager.UpdateJob(ctx, job)
		if err != nil {
			if err == db.ErrOptimisticLockError {
				continue
			}
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
					"job_id":       gid.ToGlobalID(gid.JobType, job.Metadata.ID),
					"run_id":       gid.ToGlobalID(gid.RunType, job.RunID),
					"workspace_id": gid.ToGlobalID(gid.WorkspaceType, job.WorkspaceID),
					"type":         auth.JobTokenType,
				},
			})
			if err != nil {
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

func (s *service) getNextAvailableQueuedJob(ctx context.Context, runner *models.Runner) (*models.Job, error) {
	jobSubscription := events.Subscription{
		Type:    events.JobSubscription,
		Actions: []events.SubscriptionAction{},
	}
	runnerSubscription := events.Subscription{
		Type:    events.RunnerSubscription,
		Actions: []events.SubscriptionAction{},
	}

	// Subscribe to job and run events
	subscriber := s.eventManager.Subscribe([]events.Subscription{jobSubscription, runnerSubscription})
	defer s.eventManager.Unsubscribe(subscriber)

	// Wait for next available run
	for {
		job, err := s.getNextAvailableJob(ctx, runner)
		if err != nil {
			return nil, err
		}

		if job != nil {
			return job, nil
		}

		event, err := subscriber.GetEvent(ctx)
		if err != nil {
			return nil, err
		}

		s.logger.Infof("Received %s:%s notification from db", event.Table, event.Action)
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
func (s *service) getNextAvailableJob(ctx context.Context, runner *models.Runner) (*models.Job, error) {
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

		if ws == nil {
			// This will only occur if the worspace is deleted after the job is queried
			continue
		}

		if !ws.Locked {
			// Check if this runner has priority to claim this job
			if runner.Type == models.SharedRunnerType {
				// Verify that there are no group runners available for this workspace since
				// group runners have higher precedence than shared runners
				groupRunners, err := s.dbClient.Runners.GetRunners(ctx, &db.GetRunnersInput{
					Filter: &db.RunnerFilter{
						NamespacePaths: ws.ExpandPath(),
					},
				})
				if err != nil {
					return nil, err
				}
				if len(groupRunners.Runners) != 0 {
					continue
				}
			} else {
				runnerGroupPath := runner.GetGroupPath()
				if runnerGroupPath != ws.GetGroupPath() {
					if !strings.HasPrefix(ws.GetGroupPath(), fmt.Sprintf("%s/", runnerGroupPath)) {
						continue
					}

					// Verify there are no child runners with higher precedence
					//runner.
					groupRunners, err := s.dbClient.Runners.GetRunners(ctx, &db.GetRunnersInput{
						Filter: &db.RunnerFilter{
							NamespacePaths: ws.ExpandPath(),
						},
					})
					if err != nil {
						return nil, err
					}

					runnerHasPrecedence := true
					for _, r := range groupRunners.Runners {
						if len(r.GetGroupPath()) > len(runnerGroupPath) {
							// There is a runner lower in the hieararchy which as precedence
							runnerHasPrecedence = false
							break
						}
					}

					if !runnerHasPrecedence {
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
	}
	return nil, nil
}
