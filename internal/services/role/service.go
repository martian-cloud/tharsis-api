// Package role implements the service layer functionality
// related to Tharsis roles. Roles allow a Tharsis subject
// to access resources offered by the API.
package role

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// GetRolesInput is the input for querying a list of roles.
type GetRolesInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.RoleSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Search filters role list by roleName prefix
	Search *string
}

// CreateRoleInput is the input for creating a Role.
type CreateRoleInput struct {
	Name        string
	Description string
	Permissions []models.Permission
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
	GetRoleByTRN(ctx context.Context, trn string) (*models.Role, error)
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
	ctx, span := tracer.Start(ctx, "svc.GetAvailablePermissions")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	return models.GetAssignablePermissions(), nil
}

func (s *service) GetRoleByID(ctx context.Context, id string) (*models.Role, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRoleByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	role, err := s.getRoleByID(ctx, id)
	if err != nil {
		tracing.RecordError(span, err, "failed to get role by ID")
		return nil, err
	}

	return role, nil
}

func (s *service) GetRoleByTRN(ctx context.Context, trn string) (*models.Role, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRoleByTRN")
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	role, err := s.dbClient.Roles.GetRoleByTRN(ctx, trn)
	if err != nil {
		tracing.RecordError(span, err, "failed to get role by TRN")
		return nil, err
	}

	if role == nil {
		return nil, errors.New("role with TRN %s not found", trn, errors.WithErrorCode(errors.ENotFound))
	}

	return role, nil
}

func (s *service) GetRolesByIDs(ctx context.Context, idList []string) ([]models.Role, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRolesByIDs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	result, err := s.dbClient.Roles.GetRoles(ctx, &db.GetRolesInput{
		Filter: &db.RoleFilter{
			RoleIDs: idList,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get roles")
		return nil, err
	}

	return result.Roles, nil
}

func (s *service) GetRoles(ctx context.Context, input *GetRolesInput) (*db.RolesResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRoles")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	dbInput := &db.GetRolesInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.RoleFilter{
			Search: input.Search,
		},
	}

	return s.dbClient.Roles.GetRoles(ctx, dbInput)
}

func (s *service) CreateRole(ctx context.Context, input *CreateRoleInput) (*models.Role, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateRole")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	userCaller, ok := caller.(*auth.UserCaller)
	if !ok {
		return nil, errors.New("Unsupported caller type, only users are allowed to create roles", errors.WithErrorCode(errors.EForbidden))
	}

	// Only admins are allowed to create roles
	if !userCaller.User.Admin {
		return nil, errors.New("Only system admins can create roles", errors.WithErrorCode(errors.EForbidden))
	}

	toCreate := &models.Role{
		Name:        input.Name,
		Description: input.Description,
		CreatedBy:   caller.GetSubject(),
	}

	toCreate.SetPermissions(input.Permissions)

	if err = toCreate.Validate(); err != nil {
		tracing.RecordError(span, err, "failed to validate role model")
		return nil, err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer CreateRole: %v", txErr)
		}
	}()

	createdRole, err := s.dbClient.Roles.CreateRole(txContext, toCreate)
	if err != nil {
		tracing.RecordError(span, err, "failed to create role")
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			Action:     models.ActionCreate,
			TargetType: models.TargetRole,
			TargetID:   createdRole.Metadata.ID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
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
	ctx, span := tracer.Start(ctx, "svc.UpdateRole")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	userCaller, ok := caller.(*auth.UserCaller)
	if !ok {
		return nil, errors.New("Unsupported caller type, only users are allowed to update roles", errors.WithErrorCode(errors.EForbidden))
	}

	// Only admins are allowed to update roles
	if !userCaller.User.Admin {
		return nil, errors.New("Only system admins can update roles", errors.WithErrorCode(errors.EForbidden))
	}

	if models.DefaultRoleID(input.Role.Metadata.ID).IsDefaultRole() {
		return nil, errors.New("Default roles are read-only", errors.WithErrorCode(errors.EForbidden))
	}

	if err = input.Role.Validate(); err != nil {
		tracing.RecordError(span, err, "failed to validate role model")
		return nil, err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer UpdateRole: %v", txErr)
		}
	}()

	updatedRole, err := s.dbClient.Roles.UpdateRole(txContext, input.Role)
	if err != nil {
		tracing.RecordError(span, err, "failed to update role")
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			Action:     models.ActionUpdate,
			TargetType: models.TargetRole,
			TargetID:   updatedRole.Metadata.ID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
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
	ctx, span := tracer.Start(ctx, "svc.DeleteRole")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	userCaller, ok := caller.(*auth.UserCaller)
	if !ok {
		return errors.New("Unsupported caller type, only users are allowed to delete roles", errors.WithErrorCode(errors.EForbidden))
	}

	// Only admins are allowed to delete roles
	if !userCaller.User.Admin {
		return errors.New("Only system admins can delete roles", errors.WithErrorCode(errors.EForbidden))
	}

	if models.DefaultRoleID(input.Role.Metadata.ID).IsDefaultRole() {
		return errors.New("Default roles are read-only", errors.WithErrorCode(errors.EForbidden))
	}

	// Get all the namespace memberships if any for this role.
	result, err := s.dbClient.NamespaceMemberships.GetNamespaceMemberships(ctx, &db.GetNamespaceMembershipsInput{
		Filter: &db.NamespaceMembershipFilter{
			RoleID: &input.Role.Metadata.ID, // Filter by Role's ID.
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get namespace memberships")
		return err
	}

	if !input.Force && len(result.NamespaceMemberships) > 0 {
		return errors.New(
			"This Role can't be deleted because it's currently associated with %d namespace memberships. "+
				"Setting force to true will automatically remove all associated namespace memberships.", len(result.NamespaceMemberships),
			errors.WithErrorCode(errors.EConflict),
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
		return nil, errors.New("role with id %s not found", id, errors.WithErrorCode(errors.ENotFound))
	}

	return role, nil
}
