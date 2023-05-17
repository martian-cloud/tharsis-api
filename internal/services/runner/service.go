// Package runner package
package runner

import (
	"context"
	"fmt"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// GetRunnersInput is the input for querying a list of runners
type GetRunnersInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.RunnerSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// NamespacePath is the namespace to return runners for
	NamespacePath string
	// IncludeInherited includes inherited runners in the result
	IncludeInherited bool
}

// CreateRunnerInput is the input for creating a new runner
type CreateRunnerInput struct {
	Name        string
	Description string
	GroupID     string
}

// Service implements all runner related functionality
type Service interface {
	GetRunnerByPath(ctx context.Context, path string) (*models.Runner, error)
	GetRunnerByID(ctx context.Context, id string) (*models.Runner, error)
	GetRunners(ctx context.Context, input *GetRunnersInput) (*db.RunnersResult, error)
	GetRunnersByIDs(ctx context.Context, idList []string) ([]models.Runner, error)
	CreateRunner(ctx context.Context, input *CreateRunnerInput) (*models.Runner, error)
	UpdateRunner(ctx context.Context, runner *models.Runner) (*models.Runner, error)
	DeleteRunner(ctx context.Context, runner *models.Runner) error
	AssignServiceAccountToRunner(ctx context.Context, serviceAccountID string, runnerID string) error
	UnassignServiceAccountFromRunner(ctx context.Context, serviceAccountID string, runnerID string) error
}

type service struct {
	logger          logger.Logger
	dbClient        *db.Client
	activityService activityevent.Service
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	activityService activityevent.Service,
) Service {
	return &service{
		logger:          logger,
		dbClient:        dbClient,
		activityService: activityService,
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

	err = caller.RequirePermission(ctx, permissions.ViewRunnerPermission, auth.WithNamespacePath(input.NamespacePath))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	filter := &db.RunnerFilter{}

	if input.IncludeInherited {
		pathParts := strings.Split(input.NamespacePath, "/")

		paths := []string{""}
		for len(pathParts) > 0 {
			paths = append(paths, strings.Join(pathParts, "/"))
			// Remove last element
			pathParts = pathParts[:len(pathParts)-1]
		}

		filter.NamespacePaths = paths
	} else {
		// This will return an empty result for workspace namespaces because workspaces
		// don't have runners directly associated (i.e. only group namespaces do)
		filter.NamespacePaths = []string{input.NamespacePath}
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

	namespacePaths := []string{}
	for _, r := range result.Runners {
		if r.GroupID != nil {
			namespacePaths = append(namespacePaths, r.GetGroupPath())
		}
	}

	if len(namespacePaths) > 0 {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.RunnerResourceType, auth.WithNamespacePaths(namespacePaths))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
			return nil, err
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
		err = caller.RequirePermission(ctx, permissions.DeleteRunnerPermission, auth.WithGroupID(*runner.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return err
		}
	} else {
		// Verify caller is a user.
		userCaller, ok := caller.(*auth.UserCaller)
		if !ok {
			return errors.New(
				errors.EForbidden,
				"Unsupported caller type, only users are allowed to delete shared runners",
			)
		}

		// Only admins are allowed to delete shared runners.
		if !userCaller.User.Admin {
			return errors.New(
				errors.EForbidden,
				"Only system admins can delete shared runners",
			)
		}
	}

	s.logger.Infow("Requested deletion of a runner.",
		"caller", caller.GetSubject(),
		"runnerID", runner.Metadata.ID,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer DeleteRunner: %v", txErr)
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
		return nil, errors.New(errors.ENotFound, "runner with ID %s not found", id)
	}

	if runner.GroupID != nil {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.RunnerResourceType, auth.WithGroupID(*runner.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
			return nil, err
		}
	}

	return runner, nil
}

func (s *service) GetRunnerByPath(ctx context.Context, path string) (*models.Runner, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunnerByPath")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	// Get runner from DB
	runner, err := s.dbClient.Runners.GetRunnerByPath(ctx, path)
	if err != nil {
		tracing.RecordError(span, err, "failed to get runner by path")
		return nil, err
	}

	if runner == nil {
		return nil, errors.New(errors.ENotFound, "runner with path %s not found", path)
	}

	if runner.GroupID != nil {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.RunnerResourceType, auth.WithGroupID(*runner.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
			return nil, err
		}
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
		err = caller.RequirePermission(ctx, permissions.CreateRunnerPermission, auth.WithGroupID(input.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
	} else {
		return nil, errors.New(errors.EInvalid, "shared runners can only be created via the API config")
	}

	runnerToCreate := models.Runner{
		Type:        models.GroupRunnerType,
		Name:        input.Name,
		Description: input.Description,
		GroupID:     &input.GroupID,
		CreatedBy:   caller.GetSubject(),
	}

	// Validate model
	if err = runnerToCreate.Validate(); err != nil {
		tracing.RecordError(span, err, "failed to validate runner model")
		return nil, err
	}

	s.logger.Infow("Requested creation of a runner.",
		"caller", caller.GetSubject(),
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
			s.logger.Errorf("failed to rollback tx for service layer CreateRunner: %v", txErr)
		}
	}()

	// Store runner in DB
	createdRunner, err := s.dbClient.Runners.CreateRunner(txContext, &runnerToCreate)
	if err != nil {
		tracing.RecordError(span, err, "failed to create runner")
		return nil, err
	}

	groupPath := createdRunner.GetGroupPath()

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
		err = caller.RequirePermission(ctx, permissions.UpdateRunnerPermission, auth.WithGroupID(*runner.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
	} else {
		// Verify caller is a user.
		userCaller, ok := caller.(*auth.UserCaller)
		if !ok {
			return nil, errors.New(
				errors.EForbidden,
				"Unsupported caller type, only users are allowed to update shared runners",
			)
		}

		// Only admins are allowed to update shared runners.
		if !userCaller.User.Admin {
			return nil, errors.New(
				errors.EForbidden,
				"Only system admins can update shared runners",
			)
		}
	}

	// Validate model
	if err = runner.Validate(); err != nil {
		tracing.RecordError(span, err, "failed to validate runner model")
		return nil, err
	}

	s.logger.Infow("Requested an update to a runner.",
		"caller", caller.GetSubject(),
		"runnerID", runner.Metadata.ID,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer UpdateRunner: %v", txErr)
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
		return errors.New(errors.ENotFound, "runner not found")
	}

	// Service accounts can only be assigned to group runners
	if runner.Type == models.SharedRunnerType {
		return errors.New(errors.EInvalid, "service account cannot be assigned to shared runner")
	}

	err = caller.RequirePermission(ctx, permissions.UpdateRunnerPermission, auth.WithGroupID(*runner.GroupID))
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
		return errors.New(errors.ENotFound, "service account not found")
	}

	saGroupPath := sa.GetGroupPath()
	runnerGroupPath := runner.GetGroupPath()

	// Verify that the service account is in the same group as the runner or in a parent group
	if saGroupPath != runnerGroupPath && !strings.HasPrefix(runnerGroupPath, fmt.Sprintf("%s/", saGroupPath)) {
		return errors.New(errors.EInvalid, "service account %s cannot be assigned to runner %s", sa.ResourcePath, runner.ResourcePath)
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
		return errors.New(errors.ENotFound, "runner not found")
	}

	// Service accounts can only be assigned to group runners
	if runner.Type == models.SharedRunnerType {
		return errors.New(errors.EInvalid, "service account cannot be unassigned to shared runner")
	}

	err = caller.RequirePermission(ctx, permissions.UpdateRunnerPermission, auth.WithGroupID(*runner.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	return s.dbClient.ServiceAccounts.UnassignServiceAccountFromRunner(ctx, serviceAccountID, runnerID)
}
