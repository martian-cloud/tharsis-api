// Package runner package
package runner

import (
	"context"
	"fmt"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
)

// GetRunnersInput is the input for querying a list of runners
type GetRunnersInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.RunnerSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *db.PaginationOptions
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
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err = caller.RequireAccessToNamespace(ctx, input.NamespacePath, models.ViewerRole); err != nil {
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
		return nil, err
	}
	return result, nil
}

func (s *service) GetRunnersByIDs(ctx context.Context, idList []string) ([]models.Runner, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	result, err := s.dbClient.Runners.GetRunners(ctx, &db.GetRunnersInput{
		Filter: &db.RunnerFilter{
			RunnerIDs: idList,
		},
	})
	if err != nil {
		return nil, err
	}

	groupIDs := []string{}
	for _, r := range result.Runners {
		if r.GroupID != nil {
			groupIDs = append(groupIDs, *r.GroupID)
		}
	}

	for _, groupID := range groupIDs {
		if err := caller.RequireAccessToInheritedGroupResource(ctx, groupID); err != nil {
			return nil, err
		}
	}

	return result.Runners, nil
}

func (s *service) DeleteRunner(ctx context.Context, runner *models.Runner) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	if runner.GroupID != nil {
		if rErr := caller.RequireAccessToGroup(ctx, *runner.GroupID, models.OwnerRole); rErr != nil {
			return rErr
		}
	} else {
		// Verify caller is a user.
		userCaller, ok := caller.(*auth.UserCaller)
		if !ok {
			return errors.NewError(
				errors.EForbidden,
				"Unsupported caller type, only users are allowed to delete shared runners",
			)
		}

		// Only admins are allowed to delete shared runners.
		if !userCaller.User.Admin {
			return errors.NewError(
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
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer DeleteRunner: %v", txErr)
		}
	}()

	err = s.dbClient.Runners.DeleteRunner(txContext, runner)
	if err != nil {
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
			return err
		}
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) GetRunnerByID(ctx context.Context, id string) (*models.Runner, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// Get runner from DB
	runner, err := s.dbClient.Runners.GetRunnerByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if runner == nil {
		return nil, errors.NewError(errors.ENotFound, fmt.Sprintf("runner with ID %s not found", id))
	}

	if runner.GroupID != nil {
		if err := caller.RequireAccessToInheritedGroupResource(ctx, *runner.GroupID); err != nil {
			return nil, err
		}
	}

	return runner, nil
}

func (s *service) GetRunnerByPath(ctx context.Context, path string) (*models.Runner, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// Get runner from DB
	runner, err := s.dbClient.Runners.GetRunnerByPath(ctx, path)
	if err != nil {
		return nil, err
	}

	if runner == nil {
		return nil, errors.NewError(errors.ENotFound, fmt.Sprintf("runner with path %s not found", path))
	}

	if runner.GroupID != nil {
		if err := caller.RequireAccessToInheritedGroupResource(ctx, *runner.GroupID); err != nil {
			return nil, err
		}
	}

	return runner, nil
}

func (s *service) CreateRunner(ctx context.Context, input *CreateRunnerInput) (*models.Runner, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if input.GroupID != "" {
		if err = caller.RequireAccessToGroup(ctx, input.GroupID, models.OwnerRole); err != nil {
			return nil, err
		}
	} else {
		return nil, errors.NewError(errors.EInvalid, "shared runners can only be created via the API config")
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
		return nil, err
	}

	s.logger.Infow("Requested creation of a runner.",
		"caller", caller.GetSubject(),
		"groupID", input.GroupID,
		"runnerName", input.Name,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
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
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	return createdRunner, nil
}

func (s *service) UpdateRunner(ctx context.Context, runner *models.Runner) (*models.Runner, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if runner.GroupID != nil {
		if rErr := caller.RequireAccessToGroup(ctx, *runner.GroupID, models.OwnerRole); rErr != nil {
			return nil, rErr
		}
	} else {
		// Verify caller is a user.
		userCaller, ok := caller.(*auth.UserCaller)
		if !ok {
			return nil, errors.NewError(
				errors.EForbidden,
				"Unsupported caller type, only users are allowed to update shared runners",
			)
		}

		// Only admins are allowed to update shared runners.
		if !userCaller.User.Admin {
			return nil, errors.NewError(
				errors.EForbidden,
				"Only system admins can update shared runners",
			)
		}
	}

	// Validate model
	if err = runner.Validate(); err != nil {
		return nil, err
	}

	s.logger.Infow("Requested an update to a runner.",
		"caller", caller.GetSubject(),
		"runnerID", runner.Metadata.ID,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
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
			return nil, err
		}
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	return updatedRunner, nil
}

func (s *service) AssignServiceAccountToRunner(ctx context.Context, serviceAccountID string, runnerID string) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	runner, err := s.dbClient.Runners.GetRunnerByID(ctx, runnerID)
	if err != nil {
		return err
	}

	if runner == nil {
		return errors.NewError(errors.ENotFound, "runner not found")
	}

	// Service accounts can only be assigned to group runners
	if runner.Type == models.SharedRunnerType {
		return errors.NewError(errors.EInvalid, "service account cannot be assigned to shared runner")
	}

	if rErr := caller.RequireAccessToGroup(ctx, *runner.GroupID, models.OwnerRole); rErr != nil {
		return rErr
	}

	sa, err := s.dbClient.ServiceAccounts.GetServiceAccountByID(ctx, serviceAccountID)
	if err != nil {
		return err
	}

	if sa == nil {
		return errors.NewError(errors.ENotFound, "service account not found")
	}

	saGroupPath := sa.GetGroupPath()
	runnerGroupPath := runner.GetGroupPath()

	// Verify that the service account is in the same group as the runner or in a parent group
	if saGroupPath != runnerGroupPath && !strings.HasPrefix(runnerGroupPath, fmt.Sprintf("%s/", saGroupPath)) {
		return errors.NewError(errors.EInvalid, fmt.Sprintf("service account %s cannot be assigned to runner %s", sa.ResourcePath, runner.ResourcePath))
	}

	return s.dbClient.ServiceAccounts.AssignServiceAccountToRunner(ctx, serviceAccountID, runnerID)
}

func (s *service) UnassignServiceAccountFromRunner(ctx context.Context, serviceAccountID string, runnerID string) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	runner, err := s.dbClient.Runners.GetRunnerByID(ctx, runnerID)
	if err != nil {
		return err
	}

	if runner == nil {
		return errors.NewError(errors.ENotFound, "runner not found")
	}

	// Service accounts can only be assigned to group runners
	if runner.Type == models.SharedRunnerType {
		return errors.NewError(errors.EInvalid, "service account cannot be unassigned to shared runner")
	}

	if rErr := caller.RequireAccessToGroup(ctx, *runner.GroupID, models.OwnerRole); rErr != nil {
		return rErr
	}

	return s.dbClient.ServiceAccounts.UnassignServiceAccountFromRunner(ctx, serviceAccountID, runnerID)
}
