// Package runner package
package runner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/fatih/color"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logstream"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const runnerErrorLogsBytesLimit = 2 * 1024 * 1024 // 2MiB

var runnerLogsTimestampColor = color.New(color.FgGreen)

// GetRunnersInput is the input for querying a list of runners
type GetRunnersInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.RunnerSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// NamespacePath is the namespace to return runners for
	NamespacePath *string
	// RunnerType is the type of runner to return
	RunnerType *models.RunnerType
	// IncludeInherited includes inherited runners in the result
	IncludeInherited bool
}

// GetRunnerSessionsInput is the input for querying a list of runner sessions
type GetRunnerSessionsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.RunnerSessionSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// RunnerID is the runner to return sessions for
	RunnerID string
}

// CreateRunnerInput is the input for creating a new runner
type CreateRunnerInput struct {
	GroupID         string
	Disabled        *bool
	Name            string
	Description     string
	RunUntaggedJobs bool
	Tags            []string
}

// CreateRunnerSessionInput is the input for creating a new runner session.
type CreateRunnerSessionInput struct {
	RunnerPath string
	Internal   bool
}

// SubscribeToRunnerSessionErrorLogInput includes options for setting up a log event subscription
type SubscribeToRunnerSessionErrorLogInput struct {
	LastSeenLogSize *int
	RunnerSessionID string
}

// SubscribeToRunnerSessionsInput is the input for subscribing to runner sessions
type SubscribeToRunnerSessionsInput struct {
	GroupID    *string
	RunnerID   *string
	RunnerType *models.RunnerType
}

// SessionEvent is a runner session event
type SessionEvent struct {
	RunnerSession *models.RunnerSession
	Action        string
}

// Service implements all runner related functionality
type Service interface {
	GetRunnerByID(ctx context.Context, id string) (*models.Runner, error)
	GetRunnerByTRN(ctx context.Context, trn string) (*models.Runner, error)
	GetRunners(ctx context.Context, input *GetRunnersInput) (*db.RunnersResult, error)
	GetRunnersByIDs(ctx context.Context, idList []string) ([]models.Runner, error)
	CreateRunner(ctx context.Context, input *CreateRunnerInput) (*models.Runner, error)
	UpdateRunner(ctx context.Context, runner *models.Runner) (*models.Runner, error)
	DeleteRunner(ctx context.Context, runner *models.Runner) error
	AssignServiceAccountToRunner(ctx context.Context, serviceAccountID string, runnerID string) error
	UnassignServiceAccountFromRunner(ctx context.Context, serviceAccountID string, runnerID string) error
	CreateRunnerSession(ctx context.Context, input *CreateRunnerSessionInput) (*models.RunnerSession, error)
	GetRunnerSessions(ctx context.Context, input *GetRunnerSessionsInput) (*db.RunnerSessionsResult, error)
	GetRunnerSessionByID(ctx context.Context, id string) (*models.RunnerSession, error)
	GetRunnerSessionByTRN(ctx context.Context, trn string) (*models.RunnerSession, error)
	AcceptRunnerSessionHeartbeat(ctx context.Context, sessionID string) error
	CreateRunnerSessionError(ctx context.Context, runnerSessionID string, message string) error
	ReadRunnerSessionErrorLog(ctx context.Context, runnerSessionID string, startOffset int, limit int) ([]byte, error)
	SubscribeToRunnerSessionErrorLog(ctx context.Context, options *SubscribeToRunnerSessionErrorLogInput) (<-chan *logstream.LogEvent, error)
	GetLogStreamsByRunnerSessionIDs(ctx context.Context, idList []string) ([]models.LogStream, error)
	SubscribeToRunnerSessions(ctx context.Context, options *SubscribeToRunnerSessionsInput) (<-chan *SessionEvent, error)
}

type service struct {
	logger           logger.Logger
	dbClient         *db.Client
	limitChecker     limits.LimitChecker
	activityService  activityevent.Service
	logStreamManager logstream.Manager
	eventManager     *events.EventManager
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	limitChecker limits.LimitChecker,
	activityService activityevent.Service,
	logStreamManager logstream.Manager,
	eventManager *events.EventManager,
) Service {
	// Enable timestamp color output
	runnerLogsTimestampColor.EnableColor()

	return &service{
		logger:           logger,
		dbClient:         dbClient,
		limitChecker:     limitChecker,
		activityService:  activityService,
		logStreamManager: logStreamManager,
		eventManager:     eventManager,
	}
}

func (s *service) GetRunners(ctx context.Context, input *GetRunnersInput) (*db.RunnersResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunners")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	if input.NamespacePath != nil {
		err = caller.RequirePermission(ctx, models.ViewRunnerPermission, auth.WithNamespacePath(*input.NamespacePath))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
	} else if !caller.IsAdmin() {
		// Non admin caller shouldn't be able to access all runners, so require a type.
		if input.RunnerType == nil {
			return nil, errors.New(
				"only system admins can view all runners",
				errors.WithErrorCode(errors.EForbidden),
			)
		}

		// Non admin can't retrieve runners for all groups, so require namespace path.
		if input.RunnerType.Equals(models.GroupRunnerType) {
			return nil, errors.New(
				"a namespace path is required when filtering for group runners",
				errors.WithErrorCode(errors.EInvalid),
			)
		}
	}

	filter := &db.RunnerFilter{
		RunnerType: input.RunnerType,
	}

	if input.IncludeInherited && input.NamespacePath != nil {
		pathParts := strings.Split(*input.NamespacePath, "/")

		paths := []string{""}
		for len(pathParts) > 0 {
			paths = append(paths, strings.Join(pathParts, "/"))
			// Remove last element
			pathParts = pathParts[:len(pathParts)-1]
		}

		filter.NamespacePaths = paths
	} else if input.NamespacePath != nil {
		// This will return an empty result for workspace namespaces because workspaces
		// don't have runners directly associated (i.e. only group namespaces do)
		filter.NamespacePaths = []string{*input.NamespacePath}
	}

	result, err := s.dbClient.Runners.GetRunners(ctx, &db.GetRunnersInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter:            filter,
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get runners")
		return nil, err
	}

	return result, nil
}

func (s *service) GetRunnersByIDs(ctx context.Context, idList []string) ([]models.Runner, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunnersByIDs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	result, err := s.dbClient.Runners.GetRunners(ctx, &db.GetRunnersInput{
		Filter: &db.RunnerFilter{
			RunnerIDs: idList,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get runners")
		return nil, err
	}

	for ix := range result.Runners {
		runner := result.Runners[ix]

		switch runner.Type {
		case models.GroupRunnerType:
			aErr := caller.RequireAccessToInheritableResource(ctx, types.RunnerModelType,
				auth.WithGroupID(*runner.GroupID), auth.WithRunnerID(runner.Metadata.ID))
			if aErr != nil {
				return nil, aErr
			}
		case models.SharedRunnerType:
			// Any authenticated caller can view basic runner information.
		default:
			return nil, errors.New("unknown runner type %s", runner.Type)
		}
	}

	return result.Runners, nil
}

func (s *service) DeleteRunner(ctx context.Context, runner *models.Runner) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteRunner")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	if runner.GroupID != nil {
		err = caller.RequirePermission(ctx, models.DeleteRunnerPermission, auth.WithGroupID(*runner.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return err
		}
	} else {
		// Verify caller is a user.
		userCaller, ok := caller.(*auth.UserCaller)
		if !ok {
			return errors.New(
				"Unsupported caller type, only users are allowed to delete shared runners",
				errors.WithErrorCode(errors.EForbidden))
		}

		// Only admins are allowed to delete shared runners.
		if !userCaller.User.Admin {
			return errors.New(
				"Only system admins can delete shared runners",
				errors.WithErrorCode(errors.EForbidden))
		}
	}

	s.logger.WithContextFields(ctx).Infow("Requested deletion of a runner.",
		"runnerID", runner.Metadata.ID,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer DeleteRunner: %v", txErr)
		}
	}()

	err = s.dbClient.Runners.DeleteRunner(txContext, runner)
	if err != nil {
		tracing.RecordError(span, err, "failed to delete runner")
		return err
	}

	if runner.GroupID != nil {
		groupPath := runner.GetGroupPath()

		if _, err = s.activityService.CreateActivityEvent(txContext,
			&activityevent.CreateActivityEventInput{
				NamespacePath: &groupPath,
				Action:        models.ActionDeleteChildResource,
				TargetType:    models.TargetGroup,
				TargetID:      *runner.GroupID,
				Payload: &models.ActivityEventDeleteChildResourcePayload{
					Name: runner.Name,
					ID:   runner.Metadata.ID,
					Type: string(models.TargetRunner),
				},
			}); err != nil {
			tracing.RecordError(span, err, "failed to create activity event")
			return err
		}
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) GetRunnerSessionByID(ctx context.Context, id string) (*models.RunnerSession, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunnerSessionByID")
	span.SetAttributes(attribute.String("runnerSessionID", id))
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		return nil, err
	}

	session, err := s.dbClient.RunnerSessions.GetRunnerSessionByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get runner session by ID", errors.WithSpan(span))
	}

	if session == nil {
		return nil, errors.New("runner session with ID %s not found", id, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	runner, err := s.getRunnerByID(ctx, span, session.RunnerID)
	if err != nil {
		return nil, err
	}

	if err := RequireViewerAccessToRunnerResource(ctx, runner); err != nil {
		return nil, err
	}

	return session, nil
}

func (s *service) GetRunnerSessionByTRN(ctx context.Context, trn string) (*models.RunnerSession, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunnerSessionByTRN")
	span.SetAttributes(attribute.String("runnerSessionTRN", trn))
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		return nil, err
	}

	session, err := s.dbClient.RunnerSessions.GetRunnerSessionByTRN(ctx, trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get runner session by TRN", errors.WithSpan(span))
	}

	if session == nil {
		return nil, errors.New("runner session with TRN %s not found", trn, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	runner, err := s.getRunnerByID(ctx, span, session.RunnerID)
	if err != nil {
		return nil, err
	}

	if err := RequireViewerAccessToRunnerResource(ctx, runner); err != nil {
		return nil, err
	}

	return session, nil
}

func (s *service) GetRunnerSessions(ctx context.Context, input *GetRunnerSessionsInput) (*db.RunnerSessionsResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunnerSessions")
	span.SetAttributes(attribute.String("runnerID", input.RunnerID))
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		return nil, err
	}

	runner, err := s.getRunnerByID(ctx, span, input.RunnerID)
	if err != nil {
		return nil, err
	}

	if err = RequireViewerAccessToRunnerResource(ctx, runner); err != nil {
		return nil, err
	}

	result, err := s.dbClient.RunnerSessions.GetRunnerSessions(ctx, &db.GetRunnerSessionsInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.RunnerSessionFilter{
			RunnerID: &input.RunnerID,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get runner sessions", errors.WithSpan(span))
	}

	return result, nil
}

func (s *service) AcceptRunnerSessionHeartbeat(ctx context.Context, sessionID string) error {
	ctx, span := tracer.Start(ctx, "svc.AcceptRunnerSessionHeartbeat")
	span.SetAttributes(attribute.String("sessionID", sessionID))
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	return s.dbClient.RetryOnOLE(ctx, func() error {
		session, err := s.dbClient.RunnerSessions.GetRunnerSessionByID(ctx, sessionID)
		if err != nil {
			return errors.Wrap(err, "failed to get runner session by ID", errors.WithSpan(span))
		}

		if session == nil {
			return errors.New("runner session not found", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
		}

		err = caller.RequirePermission(ctx, models.UpdateRunnerSessionPermission, auth.WithRunnerID(session.RunnerID))
		if err != nil {
			return err
		}

		session.LastContactTimestamp = time.Now().UTC()

		_, err = s.dbClient.RunnerSessions.UpdateRunnerSession(ctx, session)
		if err != nil {
			return err
		}

		return nil
	})
}

func (s *service) CreateRunnerSession(ctx context.Context, input *CreateRunnerSessionInput) (*models.RunnerSession, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateRunnerSession")
	span.SetAttributes(attribute.String("runnerPath", input.RunnerPath))
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	runner, err := s.dbClient.Runners.GetRunnerByTRN(ctx, types.RunnerModelType.BuildTRN(input.RunnerPath))
	if err != nil {
		return nil, err
	}

	if runner == nil {
		return nil, errors.New("runner not found", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	err = caller.RequirePermission(ctx, models.CreateRunnerSessionPermission, auth.WithRunnerID(runner.Metadata.ID))
	if err != nil {
		return nil, err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer CreateRunnerSession: %v", txErr)
		}
	}()

	// Execute create first to ensure that no other runner sessions can be created while the transaction is in progress
	session, err := s.dbClient.RunnerSessions.CreateRunnerSession(txContext, &models.RunnerSession{
		RunnerID:             runner.Metadata.ID,
		LastContactTimestamp: time.Now().UTC(),
		Internal:             input.Internal,
	})
	if err != nil {
		return nil, err
	}

	// Create log stream for session
	_, err = s.dbClient.LogStreams.CreateLogStream(txContext, &models.LogStream{
		RunnerSessionID: &session.Metadata.ID,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create log stream", errors.WithSpan(span))
	}

	// Check how many sessions are currently active for the runner.
	activeSessionsResponse, err := s.dbClient.RunnerSessions.GetRunnerSessions(txContext, &db.GetRunnerSessionsInput{
		Filter: &db.RunnerSessionFilter{
			RunnerID: &runner.Metadata.ID,
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get runner sessions", errors.WithSpan(span))
	}

	// Check if the sessions per runner limit has been exceeded
	if err := s.limitChecker.CheckLimit(ctx, limits.ResourceLimitRunnerSessionsPerRunner, activeSessionsResponse.PageInfo.TotalCount); err != nil {
		if errors.ErrorCode(err) != errors.EInvalid {
			return nil, errors.Wrap(err, "failed to check limit", errors.WithSpan(span))
		}

		// Remove the oldest session
		sortBy := db.RunnerSessionSortableFieldLastContactedAtAsc
		oldestSessionResponse, err := s.dbClient.RunnerSessions.GetRunnerSessions(txContext, &db.GetRunnerSessionsInput{
			Sort: &sortBy,
			Filter: &db.RunnerSessionFilter{
				RunnerID: &runner.Metadata.ID,
			},
			PaginationOptions: &pagination.Options{
				First: ptr.Int32(1),
			},
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get runner sessions", errors.WithSpan(span))
		}

		if len(oldestSessionResponse.RunnerSessions) > 0 {
			oldestSession := oldestSessionResponse.RunnerSessions[0]

			// If the oldest session is still active then no more sessions can be created until it's
			// not active anymore
			if oldestSession.Active() {
				return nil, errors.New("too many active sessions", errors.WithSpan(span), errors.WithErrorCode(errors.ETooManyRequests))
			}

			if err = s.dbClient.RunnerSessions.DeleteRunnerSession(txContext, &oldestSession); err != nil {
				return nil, err
			}
		}
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	return session, nil
}

func (s *service) GetRunnerByID(ctx context.Context, id string) (*models.Runner, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunnerByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	// Get runner from DB
	runner, err := s.dbClient.Runners.GetRunnerByID(ctx, id)
	if err != nil {
		tracing.RecordError(span, err, "failed to get runner by ID")
		return nil, err
	}

	if runner == nil {
		return nil, errors.New("runner with ID %s not found", id, errors.WithErrorCode(errors.ENotFound))
	}

	switch runner.Type {
	case models.GroupRunnerType:
		aErr := caller.RequireAccessToInheritableResource(ctx, types.RunnerModelType,
			auth.WithGroupID(*runner.GroupID), auth.WithRunnerID(runner.Metadata.ID))
		if aErr != nil {
			return nil, aErr
		}
	case models.SharedRunnerType:
		// Any authenticated caller can view basic runner information.
	default:
		return nil, errors.New("unknown runner type %s", runner.Type)
	}

	return runner, nil
}

func (s *service) GetRunnerByTRN(ctx context.Context, trn string) (*models.Runner, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunnerByTRN")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	// Get runner from DB
	runner, err := s.dbClient.Runners.GetRunnerByTRN(ctx, trn)
	if err != nil {
		tracing.RecordError(span, err, "failed to get runner by TRN")
		return nil, err
	}

	if runner == nil {
		return nil, errors.New("runner with TRN %s not found", trn, errors.WithErrorCode(errors.ENotFound))
	}

	switch runner.Type {
	case models.GroupRunnerType:
		aErr := caller.RequireAccessToInheritableResource(ctx, types.RunnerModelType,
			auth.WithGroupID(*runner.GroupID), auth.WithRunnerID(runner.Metadata.ID))
		if aErr != nil {
			return nil, aErr
		}
	case models.SharedRunnerType:
		// Any authenticated caller can view basic runner information.
	default:
		return nil, errors.New("unknown runner type %s", runner.Type)
	}

	return runner, nil
}

func (s *service) CreateRunner(ctx context.Context, input *CreateRunnerInput) (*models.Runner, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateRunner")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	if input.GroupID != "" {
		err = caller.RequirePermission(ctx, models.CreateRunnerPermission, auth.WithGroupID(input.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
	} else {
		return nil, errors.New("shared runners can only be created via the API config", errors.WithErrorCode(errors.EInvalid))
	}

	runnerToCreate := models.Runner{
		Type:            models.GroupRunnerType,
		Name:            input.Name,
		Description:     input.Description,
		GroupID:         &input.GroupID,
		CreatedBy:       caller.GetSubject(),
		Tags:            input.Tags,
		RunUntaggedJobs: input.RunUntaggedJobs,
	}

	// Validate model
	if err = runnerToCreate.Validate(); err != nil {
		tracing.RecordError(span, err, "failed to validate runner model")
		return nil, err
	}

	s.logger.WithContextFields(ctx).Infow("Requested creation of a runner.",
		"groupID", input.GroupID,
		"runnerName", input.Name,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer CreateRunner: %v", txErr)
		}
	}()

	// Store runner in DB
	createdRunner, err := s.dbClient.Runners.CreateRunner(txContext, &runnerToCreate)
	if err != nil {
		tracing.RecordError(span, err, "failed to create runner")
		return nil, err
	}

	groupPath := createdRunner.GetGroupPath()

	// Get the number of runners for the group to check whether we just violated the limit.
	newRunners, err := s.dbClient.Runners.GetRunners(txContext, &db.GetRunnersInput{
		Filter: &db.RunnerFilter{
			NamespacePaths: []string{groupPath},
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get group's runners")
		return nil, err
	}
	if err = s.limitChecker.CheckLimit(txContext, limits.ResourceLimitRunnerAgentsPerGroup, newRunners.PageInfo.TotalCount); err != nil {
		tracing.RecordError(span, err, "limit check failed")
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetRunner,
			TargetID:      createdRunner.Metadata.ID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	return createdRunner, nil
}

// UpdateRunner updates a runner.
// In Tharsis, the model passed as an argument already has the 'Disabled' field set to the desired value.
func (s *service) UpdateRunner(ctx context.Context, runner *models.Runner) (*models.Runner, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateRunner")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	if runner.GroupID != nil {
		err = caller.RequirePermission(ctx, models.UpdateRunnerPermission, auth.WithGroupID(*runner.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
	} else {
		// Verify caller is a user.
		userCaller, ok := caller.(*auth.UserCaller)
		if !ok {
			return nil, errors.New(
				"Unsupported caller type, only users are allowed to update shared runners",
				errors.WithErrorCode(errors.EForbidden))
		}

		// Only admins are allowed to update shared runners.
		if !userCaller.User.Admin {
			return nil, errors.New(
				"Only system admins can update shared runners",
				errors.WithErrorCode(errors.EForbidden))
		}
	}

	// Validate model
	if err = runner.Validate(); err != nil {
		tracing.RecordError(span, err, "failed to validate runner model")
		return nil, err
	}

	s.logger.WithContextFields(ctx).Infow("Requested an update to a runner.",
		"runnerID", runner.Metadata.ID,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer UpdateRunner: %v", txErr)
		}
	}()

	// Store runner in DB
	updatedRunner, err := s.dbClient.Runners.UpdateRunner(txContext, runner)
	if err != nil {
		tracing.RecordError(span, err, "failed to update runner")
		return nil, err
	}

	if runner.GroupID != nil {
		groupPath := updatedRunner.GetGroupPath()

		if _, err = s.activityService.CreateActivityEvent(txContext,
			&activityevent.CreateActivityEventInput{
				NamespacePath: &groupPath,
				Action:        models.ActionUpdate,
				TargetType:    models.TargetRunner,
				TargetID:      updatedRunner.Metadata.ID,
			}); err != nil {
			tracing.RecordError(span, err, "failed to create activity event")
			return nil, err
		}
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	return updatedRunner, nil
}

func (s *service) AssignServiceAccountToRunner(ctx context.Context, serviceAccountID string, runnerID string) error {
	ctx, span := tracer.Start(ctx, "svc.AssignServiceAccountToRunner")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	runner, err := s.dbClient.Runners.GetRunnerByID(ctx, runnerID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get runner by ID")
		return err
	}

	if runner == nil {
		return errors.New("runner not found", errors.WithErrorCode(errors.ENotFound))
	}

	// Service accounts can only be assigned to group runners
	if runner.Type == models.SharedRunnerType {
		return errors.New("service account cannot be assigned to shared runner", errors.WithErrorCode(errors.EInvalid))
	}

	err = caller.RequirePermission(ctx, models.UpdateRunnerPermission, auth.WithGroupID(*runner.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	sa, err := s.dbClient.ServiceAccounts.GetServiceAccountByID(ctx, serviceAccountID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get service account by ID")
		return err
	}

	if sa == nil {
		return errors.New("service account not found", errors.WithErrorCode(errors.ENotFound))
	}

	saGroupPath := sa.GetGroupPath()
	runnerGroupPath := runner.GetGroupPath()

	// Verify that the service account is in the same group as the runner or in a parent group
	if saGroupPath != runnerGroupPath && !strings.HasPrefix(runnerGroupPath, fmt.Sprintf("%s/", saGroupPath)) {
		return errors.New("service account %s cannot be assigned to runner %s", sa.GetResourcePath(), runner.GetResourcePath(), errors.WithErrorCode(errors.EInvalid))
	}

	return s.dbClient.ServiceAccounts.AssignServiceAccountToRunner(ctx, serviceAccountID, runnerID)
}

func (s *service) UnassignServiceAccountFromRunner(ctx context.Context, serviceAccountID string, runnerID string) error {
	ctx, span := tracer.Start(ctx, "svc.UnassignServiceAccountFromRunner")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	runner, err := s.dbClient.Runners.GetRunnerByID(ctx, runnerID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get runner by ID")
		return err
	}

	if runner == nil {
		return errors.New("runner not found", errors.WithErrorCode(errors.ENotFound))
	}

	// Service accounts can only be assigned to group runners
	if runner.Type == models.SharedRunnerType {
		return errors.New("service account cannot be unassigned to shared runner", errors.WithErrorCode(errors.EInvalid))
	}

	err = caller.RequirePermission(ctx, models.UpdateRunnerPermission, auth.WithGroupID(*runner.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	return s.dbClient.ServiceAccounts.UnassignServiceAccountFromRunner(ctx, serviceAccountID, runnerID)
}

func (s *service) CreateRunnerSessionError(ctx context.Context, runnerSessionID string, message string) error {
	ctx, span := tracer.Start(ctx, "svc.CreateRunnerSessionError")
	span.SetAttributes(attribute.String("runner_session_id", runnerSessionID))
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	return s.dbClient.RetryOnOLE(ctx, func() error {
		session, err := s.getRunnerSessionByID(ctx, span, runnerSessionID)
		if err != nil {
			return err
		}

		err = caller.RequirePermission(ctx, models.UpdateRunnerSessionPermission, auth.WithRunnerID(session.RunnerID))
		if err != nil {
			return err
		}

		// Find log stream associated with runner session
		stream, err := s.dbClient.LogStreams.GetLogStreamByRunnerSessionID(ctx, runnerSessionID)
		if err != nil {
			return err
		}
		if stream == nil {
			return errors.New("log stream not found for runner session: %s", runnerSessionID)
		}

		timestamp := runnerLogsTimestampColor.Sprintf("%s:", time.Now().UTC().Format(time.RFC3339Nano))

		buf := []byte(fmt.Sprintf("%s %s\n", timestamp, message))

		// Check if the new logs will exceed the limit
		if (stream.Size + len(buf)) > runnerErrorLogsBytesLimit {
			return errors.New("runner session error log size limit exceeded", errors.WithErrorCode(errors.ETooLarge))
		}

		txContext, err := s.dbClient.Transactions.BeginTx(ctx)
		if err != nil {
			return err
		}

		defer func() {
			if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
				s.logger.WithContextFields(ctx).Errorf("failed to rollback tx: %v", txErr)
			}
		}()

		// Update session error count
		session.ErrorCount++
		_, err = s.dbClient.RunnerSessions.UpdateRunnerSession(txContext, session)
		if err != nil {
			return err
		}

		// Write logs to store
		_, err = s.logStreamManager.WriteLogs(txContext, stream.Metadata.ID, stream.Size, buf)
		if err != nil {
			return err
		}

		return s.dbClient.Transactions.CommitTx(txContext)
	})
}

func (s *service) ReadRunnerSessionErrorLog(ctx context.Context, runnerSessionID string, startOffset int, limit int) ([]byte, error) {
	ctx, span := tracer.Start(ctx, "svc.ReadRunnerSessionErrorLog")
	span.SetAttributes(attribute.String("runner_session_id", runnerSessionID))
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		return nil, err
	}

	session, err := s.getRunnerSessionByID(ctx, span, runnerSessionID)
	if err != nil {
		return nil, err
	}

	runner, err := s.getRunnerByID(ctx, span, session.RunnerID)
	if err != nil {
		return nil, err
	}

	if err = RequireViewerAccessToRunnerResource(ctx, runner); err != nil {
		return nil, err
	}

	// Find log stream associated with runner session
	stream, err := s.dbClient.LogStreams.GetLogStreamByRunnerSessionID(ctx, runnerSessionID)
	if err != nil {
		return nil, err
	}
	if stream == nil {
		return nil, errors.New("log stream not found for runner session: %s", runnerSessionID)
	}

	return s.logStreamManager.ReadLogs(ctx, stream.Metadata.ID, startOffset, limit)
}

func (s *service) SubscribeToRunnerSessionErrorLog(ctx context.Context, options *SubscribeToRunnerSessionErrorLogInput) (<-chan *logstream.LogEvent, error) {
	ctx, span := tracer.Start(ctx, "svc.SubscribeToRunnerSessionErrorLog")
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		return nil, err
	}

	session, err := s.getRunnerSessionByID(ctx, span, options.RunnerSessionID)
	if err != nil {
		return nil, err
	}

	runner, err := s.getRunnerByID(ctx, span, session.RunnerID)
	if err != nil {
		return nil, err
	}

	if err = RequireViewerAccessToRunnerResource(ctx, runner); err != nil {
		return nil, err
	}

	logStream, err := s.dbClient.LogStreams.GetLogStreamByRunnerSessionID(ctx, session.Metadata.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get log stream by runner session ID", errors.WithSpan(span))
	}

	if logStream == nil {
		return nil, fmt.Errorf("log stream not found for runner session %s", session.Metadata.ID)
	}

	return s.logStreamManager.Subscribe(ctx, &logstream.SubscriptionOptions{
		LastSeenLogSize: options.LastSeenLogSize,
		LogStreamID:     logStream.Metadata.ID,
	})
}

func (s *service) GetLogStreamsByRunnerSessionIDs(ctx context.Context, idList []string) ([]models.LogStream, error) {
	ctx, span := tracer.Start(ctx, "svc.GetLogStreamsByRunnerSessionIDs")
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		return nil, err
	}

	if len(idList) == 0 {
		return []models.LogStream{}, nil
	}

	runnerSessionsResp, err := s.dbClient.RunnerSessions.GetRunnerSessions(ctx, &db.GetRunnerSessionsInput{
		Filter: &db.RunnerSessionFilter{
			RunnerSessionIDs: idList,
		},
	})
	if err != nil {
		return nil, err
	}

	runnerIDMap := map[string]struct{}{}
	for _, session := range runnerSessionsResp.RunnerSessions {
		runnerIDMap[session.RunnerID] = struct{}{}
	}

	runnerIDs := make([]string, 0, len(runnerIDMap))
	for runnerID := range runnerIDMap {
		runnerIDs = append(runnerIDs, runnerID)
	}

	runnersResp, err := s.dbClient.Runners.GetRunners(ctx, &db.GetRunnersInput{
		Filter: &db.RunnerFilter{
			RunnerIDs: runnerIDs,
		},
	})
	if err != nil {
		return nil, err
	}

	// Verify caller has access to all runners.
	for ix := range runnersResp.Runners {
		if err = RequireViewerAccessToRunnerResource(ctx, &runnersResp.Runners[ix]); err != nil {
			return nil, err
		}
	}

	result, err := s.dbClient.LogStreams.GetLogStreams(ctx, &db.GetLogStreamsInput{
		Filter: &db.LogStreamFilter{
			RunnerSessionIDs: idList,
		},
	})
	if err != nil {
		return nil, err
	}

	return result.LogStreams, nil
}

func (s *service) SubscribeToRunnerSessions(ctx context.Context, options *SubscribeToRunnerSessionsInput) (<-chan *SessionEvent, error) {
	ctx, span := tracer.Start(ctx, "svc.SubscribeToRunnerSessions")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if options.GroupID != nil {
		err = caller.RequireAccessToInheritableResource(ctx, types.RunnerModelType, auth.WithGroupID(*options.GroupID))
		if err != nil {
			return nil, err
		}
	} else if options.RunnerID != nil {
		runner, err := s.getRunnerByID(ctx, span, *options.RunnerID)
		if err != nil {
			return nil, err
		}
		if err = RequireViewerAccessToRunnerResource(ctx, runner); err != nil {
			return nil, err
		}
	} else if !caller.IsAdmin() {
		return nil, errors.New(
			"Only system admins can subscribe to all runner sessions",
			errors.WithErrorCode(errors.EForbidden),
		)
	}

	subscription := events.Subscription{
		Type: events.RunnerSessionSubscription,
		Actions: []events.SubscriptionAction{
			events.CreateAction,
			events.UpdateAction,
		},
	}

	subscriber := s.eventManager.Subscribe([]events.Subscription{subscription})

	outgoing := make(chan *SessionEvent)
	go func() {
		// Defer close of outgoing channel
		defer close(outgoing)
		defer s.eventManager.Unsubscribe(subscriber)

		// Wait for runner session updates
		for {
			event, err := subscriber.GetEvent(ctx)
			if err != nil {
				if !errors.IsContextCanceledError(err) && !errors.IsDeadlineExceededError(err) {
					s.logger.WithContextFields(ctx).Errorf("error occurred while waiting for runner session events: %v", err)
				}
				return
			}

			eventData, err := event.ToRunnerSessionEventData()
			if err != nil {
				s.logger.WithContextFields(ctx).Errorf("failed to get runner session event data in run session event subscription: %v", err)
				continue
			}

			// Check if this event is for the runner we're interested in
			if options.RunnerID != nil && *options.RunnerID != eventData.RunnerID {
				continue
			}

			if options.GroupID != nil || options.RunnerID != nil {
				// We need to query the runner to check if it belongs to the organization
				runner, err := s.getRunnerByID(ctx, span, eventData.RunnerID)
				if err != nil {
					s.logger.WithContextFields(ctx).Errorf("error querying for runner in subscription goroutine: %v", err)
					continue
				}
				if options.GroupID != nil && *runner.GroupID != *options.GroupID {
					continue
				}
				if options.RunnerType != nil && runner.Type != *options.RunnerType {
					continue
				}
			}

			session, err := s.dbClient.RunnerSessions.GetRunnerSessionByID(ctx, event.ID)
			if err != nil {
				s.logger.WithContextFields(ctx).Errorf("error querying for runner session in subscription goroutine: %v", err)
				continue
			}
			if session == nil {
				s.logger.WithContextFields(ctx).Errorf("Received event for runner session that does not exist %s", event.ID)
				continue
			}

			select {
			case <-ctx.Done():
				return
			case outgoing <- &SessionEvent{RunnerSession: session, Action: event.Action}:
			}
		}
	}()

	return outgoing, nil
}

func (s *service) getRunnerByID(ctx context.Context, span trace.Span, id string) (*models.Runner, error) {
	runner, err := s.dbClient.Runners.GetRunnerByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get runner by ID", errors.WithSpan(span))
	}

	if runner == nil {
		return nil, errors.New("runner with ID %s not found", id, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	return runner, nil
}

func (s *service) getRunnerSessionByID(ctx context.Context, span trace.Span, id string) (*models.RunnerSession, error) {
	session, err := s.dbClient.RunnerSessions.GetRunnerSessionByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get runner session by ID", errors.WithSpan(span))
	}

	if session == nil {
		return nil, errors.New("runner session with ID %s not found", id, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	return session, nil
}

// RequireViewerAccessToRunnerResource checks if the caller has viewer access to a runner's
// resource like sessions, jobs, logs, etc.
func RequireViewerAccessToRunnerResource(ctx context.Context, runner *models.Runner) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	switch runner.Type {
	case models.GroupRunnerType:
		err := caller.RequireAccessToInheritableResource(ctx, types.RunnerModelType,
			auth.WithGroupID(*runner.GroupID), auth.WithRunnerID(runner.Metadata.ID))
		if err != nil {
			return err
		}
	case models.SharedRunnerType:
		if !caller.IsAdmin() {
			return errors.New(
				"Only system admins can access shared runner's resources like jobs, sessions and logs",
				errors.WithErrorCode(errors.EForbidden),
			)
		}
	default:
		return errors.New("unknown runner type %s", runner.Type)
	}

	return nil
}
