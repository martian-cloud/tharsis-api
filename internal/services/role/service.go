// Package role implements the service layer functionality
// related to Tharsis roles. Roles allow a Tharsis subject
// to access resources offered by the API.
package role

import (
	"context"
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// GetRolesInput is the input for querying a list of roles.
type GetRolesInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.RoleSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// RoleNamePrefix filters role list by roleName prefix
	RoleNamePrefix *string
}

// CreateRoleInput is the input for creating a Role.
type CreateRoleInput struct {
	Name        string
	Description string
	Permissions []permissions.Permission
}

// UpdateRoleInput is the input for updating a Role.
type UpdateRoleInput struct {
	Role *models.Role
}

// DeleteRoleInput is the input for deleting a Role.
type DeleteRoleInput struct {
	Role  *models.Role
	Force bool
}

// Service implements all the functionality related to Roles.
type Service interface {
	GetAvailablePermissions(ctx context.Context) ([]string, error)
	GetRoleByID(ctx context.Context, id string) (*models.Role, error)
	GetRoleByName(ctx context.Context, name string) (*models.Role, error)
	GetRolesByIDs(ctx context.Context, idList []string) ([]models.Role, error)
	GetRoles(ctx context.Context, input *GetRolesInput) (*db.RolesResult, error)
	CreateRole(ctx context.Context, input *CreateRoleInput) (*models.Role, error)
	UpdateRole(ctx context.Context, input *UpdateRoleInput) (*models.Role, error)
	DeleteRole(ctx context.Context, input *DeleteRoleInput) error
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

func (s *service) GetAvailablePermissions(ctx context.Context) ([]string, error) {
	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		return nil, err
	}

	return permissions.GetAssignablePermissions(), nil
}

func (s *service) GetRoleByID(ctx context.Context, id string) (*models.Role, error) {
	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		return nil, err
	}

	role, err := s.getRoleByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return role, nil
}

func (s *service) GetRoleByName(ctx context.Context, name string) (*models.Role, error) {
	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		return nil, err
	}

	role, err := s.dbClient.Roles.GetRoleByName(ctx, name)
	if err != nil {
		return nil, err
	}

	if role == nil {
		return nil, errors.NewError(errors.ENotFound, fmt.Sprintf("Role with name %s not found", name))
	}

	return role, nil
}

func (s *service) GetRolesByIDs(ctx context.Context, idList []string) ([]models.Role, error) {
	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		return nil, err
	}

	result, err := s.dbClient.Roles.GetRoles(ctx, &db.GetRolesInput{
		Filter: &db.RoleFilter{
			RoleIDs: idList,
		},
	})
	if err != nil {
		return nil, err
	}

	return result.Roles, nil
}

func (s *service) GetRoles(ctx context.Context, input *GetRolesInput) (*db.RolesResult, error) {
	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		return nil, err
	}

	dbInput := &db.GetRolesInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.RoleFilter{
			RoleNamePrefix: input.RoleNamePrefix,
		},
	}

	return s.dbClient.Roles.GetRoles(ctx, dbInput)
}

func (s *service) CreateRole(ctx context.Context, input *CreateRoleInput) (*models.Role, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	userCaller, ok := caller.(*auth.UserCaller)
	if !ok {
		return nil, errors.NewError(errors.EForbidden, "Unsupported caller type, only users are allowed to create roles")
	}

	// Only admins are allowed to create roles
	if !userCaller.User.Admin {
		return nil, errors.NewError(errors.EForbidden, "Only system admins can create roles")
	}

	toCreate := &models.Role{
		Name:        input.Name,
		Description: input.Description,
		CreatedBy:   caller.GetSubject(),
	}

	toCreate.SetPermissions(input.Permissions)

	if err = toCreate.Validate(); err != nil {
		return nil, err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer CreateRole: %v", txErr)
		}
	}()

	createdRole, err := s.dbClient.Roles.CreateRole(txContext, toCreate)
	if err != nil {
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			Action:     models.ActionCreate,
			TargetType: models.TargetRole,
			TargetID:   createdRole.Metadata.ID,
		}); err != nil {
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	s.logger.Infow("Created a role.",
		"caller", caller.GetSubject(),
		"roleName", input.Name,
		"roleID", createdRole.Metadata.ID,
	)

	return createdRole, nil
}

func (s *service) UpdateRole(ctx context.Context, input *UpdateRoleInput) (*models.Role, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	userCaller, ok := caller.(*auth.UserCaller)
	if !ok {
		return nil, errors.NewError(errors.EForbidden, "Unsupported caller type, only users are allowed to update roles")
	}

	// Only admins are allowed to update roles
	if !userCaller.User.Admin {
		return nil, errors.NewError(errors.EForbidden, "Only system admins can update roles")
	}

	if models.DefaultRoleID(input.Role.Metadata.ID).IsDefaultRole() {
		return nil, errors.NewError(errors.EForbidden, "Default roles are read-only")
	}

	if err = input.Role.Validate(); err != nil {
		return nil, err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer UpdateRole: %v", txErr)
		}
	}()

	updatedRole, err := s.dbClient.Roles.UpdateRole(txContext, input.Role)
	if err != nil {
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			Action:     models.ActionUpdate,
			TargetType: models.TargetRole,
			TargetID:   updatedRole.Metadata.ID,
		}); err != nil {
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	s.logger.Infow("Updated a role.",
		"caller", caller.GetSubject(),
		"roleName", input.Role.Name,
		"roleID", input.Role.Metadata.ID,
	)

	return updatedRole, nil
}

func (s *service) DeleteRole(ctx context.Context, input *DeleteRoleInput) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	userCaller, ok := caller.(*auth.UserCaller)
	if !ok {
		return errors.NewError(errors.EForbidden, "Unsupported caller type, only users are allowed to delete roles")
	}

	// Only admins are allowed to delete roles
	if !userCaller.User.Admin {
		return errors.NewError(errors.EForbidden, "Only system admins can delete roles")
	}

	if models.DefaultRoleID(input.Role.Metadata.ID).IsDefaultRole() {
		return errors.NewError(errors.EForbidden, "Default roles are read-only")
	}

	// Get all the namespace memberships if any for this role.
	result, err := s.dbClient.NamespaceMemberships.GetNamespaceMemberships(ctx, &db.GetNamespaceMembershipsInput{
		Filter: &db.NamespaceMembershipFilter{
			RoleID: &input.Role.Metadata.ID, // Filter by Role's ID.
		},
	})
	if err != nil {
		return err
	}

	if !input.Force && len(result.NamespaceMemberships) > 0 {
		return errors.NewError(
			errors.EConflict,
			fmt.Sprintf("This Role can't be deleted because it's currently associated with %d namespace memberships. "+
				"Setting force to true will automatically remove all associated namespace memberships.", len(result.NamespaceMemberships)),
		)
	}

	s.logger.Infow("Requested to delete a role.",
		"caller", caller.GetSubject(),
		"roleName", input.Role.Name,
		"roleID", input.Role.Metadata.ID,
	)

	return s.dbClient.Roles.DeleteRole(ctx, input.Role)
}

func (s *service) getRoleByID(ctx context.Context, id string) (*models.Role, error) {
	role, err := s.dbClient.Roles.GetRoleByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if role == nil {
		return nil, errors.NewError(errors.ENotFound, fmt.Sprintf("Role with id %s not found", id))
	}

	return role, nil
}
