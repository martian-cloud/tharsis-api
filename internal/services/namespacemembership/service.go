// Package namespacemembership package
package namespacemembership

//go:generate go tool mockery --name Service --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace/utils"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// CreateNamespaceMembershipInput is the input for creating a new namespace membership
type CreateNamespaceMembershipInput struct {
	User           *models.User
	ServiceAccount *models.ServiceAccount
	Team           *models.Team
	RoleID         string
	NamespacePath  string
}

// GetNamespaceMembershipsForSubjectInput is the input for querying a list of namespace memberships
type GetNamespaceMembershipsForSubjectInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.NamespaceMembershipSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// UserID filters the namespace memberships by user ID
	UserID *string
	// ServiceAccount filters the namespace memberships by this service account
	ServiceAccount *models.ServiceAccount
}

// Service implements all namespace membership related functionality
type Service interface {
	GetNamespaceMembershipsForNamespace(ctx context.Context, namespacePath string) ([]models.NamespaceMembership, error)
	GetNamespaceMembershipsForSubject(ctx context.Context, input *GetNamespaceMembershipsForSubjectInput) (*db.NamespaceMembershipResult, error)
	GetNamespaceMembershipByID(ctx context.Context, id string) (*models.NamespaceMembership, error)
	GetNamespaceMembershipsByIDs(ctx context.Context, ids []string) ([]models.NamespaceMembership, error)
	CreateNamespaceMembership(ctx context.Context, input *CreateNamespaceMembershipInput) (*models.NamespaceMembership, error)
	UpdateNamespaceMembership(ctx context.Context, namespaceMembership *models.NamespaceMembership) (*models.NamespaceMembership, error)
	DeleteNamespaceMembership(ctx context.Context, namespaceMembership *models.NamespaceMembership) error
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

func (s *service) GetNamespaceMembershipsForNamespace(ctx context.Context, namespacePath string) ([]models.NamespaceMembership, error) {
	ctx, span := tracer.Start(ctx, "svc.GetNamespaceMembershipsForNamespace")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.ViewNamespaceMembershipPermission, auth.WithNamespacePath(namespacePath))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	pathParts := strings.Split(namespacePath, "/")

	paths := []string{}
	for len(pathParts) > 0 {
		paths = append(paths, strings.Join(pathParts, "/"))
		// Remove last element
		pathParts = pathParts[:len(pathParts)-1]
	}

	sort := db.NamespaceMembershipSortableFieldNamespacePathDesc
	dbInput := &db.GetNamespaceMembershipsInput{
		Sort: &sort,
		Filter: &db.NamespaceMembershipFilter{
			NamespacePaths: paths,
		},
	}

	result, err := s.dbClient.NamespaceMemberships.GetNamespaceMemberships(ctx, dbInput)
	if err != nil {
		tracing.RecordError(span, err, "failed to get namespace memberships")
		return nil, err
	}

	namespaceMemberships := []models.NamespaceMembership{}

	seen := map[string]bool{}
	for _, m := range result.NamespaceMemberships {
		var keyAndCategory string
		// Exactly one of these should take effect.
		switch {
		case m.UserID != nil:
			keyAndCategory = fmt.Sprintf("user::%s", *m.UserID)
		case m.ServiceAccountID != nil:
			keyAndCategory = fmt.Sprintf("service-account::%s", *m.ServiceAccountID)
		case m.TeamID != nil:
			keyAndCategory = fmt.Sprintf("team::%s", *m.TeamID)
		}

		if _, ok := seen[keyAndCategory]; !ok {
			namespaceMemberships = append(namespaceMemberships, m)

			seen[keyAndCategory] = true
		}
	}

	return namespaceMemberships, nil
}

func (s *service) GetNamespaceMembershipsForSubject(ctx context.Context,
	input *GetNamespaceMembershipsForSubjectInput,
) (*db.NamespaceMembershipResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetNamespaceMembershipsForSubject")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	// Exactly one of these should take effect.
	switch {
	case input.UserID != nil:
		userCaller, ok := caller.(*auth.UserCaller)
		if !ok || (!userCaller.User.Admin && userCaller.User.Metadata.ID != *input.UserID) {
			return nil, errors.New("User %s is not authorized to query namespace memberships for %s", userCaller.User.Username, *input.UserID, errors.WithErrorCode(errors.EForbidden))
		}
	case input.ServiceAccount != nil:
		// Verify caller has access to the group this service account is in.
		err = caller.RequirePermission(ctx, permissions.ViewNamespaceMembershipPermission, auth.WithGroupID(input.ServiceAccount.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
	default:
		return nil, errors.New("input is missing required fields", errors.WithErrorCode(errors.EInvalid))
	}

	dbInput := &db.GetNamespaceMembershipsInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.NamespaceMembershipFilter{
			UserID: input.UserID,
		},
	}

	if input.ServiceAccount != nil {
		dbInput.Filter.ServiceAccountID = &input.ServiceAccount.Metadata.ID
	}

	return s.dbClient.NamespaceMemberships.GetNamespaceMemberships(ctx, dbInput)
}

func (s *service) GetNamespaceMembershipByID(ctx context.Context, id string) (*models.NamespaceMembership, error) {
	ctx, span := tracer.Start(ctx, "svc.GetNamespaceMembershipByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	namespaceMembership, err := s.dbClient.NamespaceMemberships.GetNamespaceMembershipByID(ctx, id)
	if err != nil {
		tracing.RecordError(span, err, "failed to get namespace membership by ID")
		return nil, err
	}

	if namespaceMembership == nil {
		return nil, errors.New("namespace membership with id %s not found", id, errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequirePermission(ctx, permissions.ViewNamespaceMembershipPermission, auth.WithNamespacePath(namespaceMembership.Namespace.Path))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return namespaceMembership, nil
}

func (s *service) GetNamespaceMembershipsByIDs(ctx context.Context, ids []string) ([]models.NamespaceMembership, error) {
	ctx, span := tracer.Start(ctx, "svc.GetNamespaceMembershipsByIDs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	// Get memberships from DB.
	resp, err := s.dbClient.NamespaceMemberships.GetNamespaceMemberships(ctx,
		&db.GetNamespaceMembershipsInput{
			Filter: &db.NamespaceMembershipFilter{
				NamespaceMembershipIDs: ids,
			},
		})
	if err != nil {
		tracing.RecordError(span, err, "failed to get namespace memberships")
		return nil, err
	}

	namespacePaths := []string{}
	for _, namespaceMembership := range resp.NamespaceMemberships {
		namespacePaths = append(namespacePaths, namespaceMembership.Namespace.Path)
	}

	if len(namespacePaths) > 0 {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.NamespaceMembershipResourceType, auth.WithNamespacePaths(namespacePaths))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
			return nil, err
		}
	}

	return resp.NamespaceMemberships, nil
}

func (s *service) CreateNamespaceMembership(ctx context.Context,
	input *CreateNamespaceMembershipInput,
) (*models.NamespaceMembership, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateNamespaceMembership")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	err := s.requirePermissionForNamespace(ctx, input.NamespacePath, permissions.CreateNamespaceMembershipPermission)
	if err != nil {
		tracing.RecordError(span, err, "namespace permission check failed")
		return nil, err
	}

	// Exactly one of user, service account, and team must be specified.
	count := 0
	if input.User != nil {
		count++
	}
	if input.ServiceAccount != nil {
		count++
	}
	if input.Team != nil {
		count++
	}
	if count != 1 {
		return nil, errors.New("Exactly one of User, ServiceAccount, team field must be defined", errors.WithErrorCode(errors.EInvalid))
	}

	// If this is a service account, we need to verify that it's being added to the group that it is associated with
	// or a nested group
	if input.ServiceAccount != nil {
		// Remove service account name from resource path
		parts := strings.Split(input.ServiceAccount.ResourcePath, "/")
		serviceAccountNamespace := strings.Join(parts[:len(parts)-1], "/")

		if serviceAccountNamespace != input.NamespacePath && !utils.IsDescendantOfPath(input.NamespacePath, serviceAccountNamespace) {
			return nil, errors.New(
				"Service account cannot be added as a member to group %s because it doesn't exist in the group or a parent group",
				input.NamespacePath,
				errors.WithErrorCode(errors.EInvalid),
			)
		}
	}

	var userID, serviceAccountID, teamID *string
	if input.User != nil {
		userID = &input.User.Metadata.ID
	}
	if input.ServiceAccount != nil {
		serviceAccountID = &input.ServiceAccount.Metadata.ID
	}
	if input.Team != nil {
		teamID = &input.Team.Metadata.ID
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer CreateNamespaceMembership: %v", txErr)
		}
	}()

	namespaceMembership, err := s.dbClient.NamespaceMemberships.CreateNamespaceMembership(txContext,
		&db.CreateNamespaceMembershipInput{
			NamespacePath:    input.NamespacePath,
			RoleID:           input.RoleID,
			UserID:           userID,
			ServiceAccountID: serviceAccountID,
			TeamID:           teamID,
		})
	if err != nil {
		tracing.RecordError(span, err, "failed to create namespace membership")
		return nil, err
	}

	// Find the role name.
	role, err := s.getRoleByID(ctx, input.RoleID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get role by ID")
		return nil, err
	}

	eventTargetType, eventTargetID := getTargetTypeID(namespaceMembership)

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &input.NamespacePath,
			Action:        models.ActionCreateMembership,
			TargetType:    eventTargetType,
			TargetID:      eventTargetID,
			Payload: &models.ActivityEventCreateNamespaceMembershipPayload{
				UserID:           namespaceMembership.UserID,
				ServiceAccountID: namespaceMembership.ServiceAccountID,
				TeamID:           namespaceMembership.TeamID,
				Role:             string(role.Name),
			},
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	return namespaceMembership, nil
}

func (s *service) UpdateNamespaceMembership(ctx context.Context,
	namespaceMembership *models.NamespaceMembership,
) (*models.NamespaceMembership, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateNamespaceMembership")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	err := s.requirePermissionForNamespace(ctx, namespaceMembership.Namespace.Path, permissions.UpdateNamespaceMembershipPermission)
	if err != nil {
		tracing.RecordError(span, err, "namespace permission check failed")
		return nil, err
	}

	// Get current state of namespace membership
	currentMembership, err := s.dbClient.NamespaceMemberships.GetNamespaceMembershipByID(ctx, namespaceMembership.Metadata.ID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get namespace membership by ID")
		return nil, err
	}

	if currentMembership == nil {
		return nil, errors.New("namespace membership with ID %s not found", namespaceMembership.Metadata.ID, errors.WithErrorCode(errors.ENotFound))
	}

	if currentMembership.RoleID == namespaceMembership.RoleID {
		// Noop if role being changed to is the same as current role.
		return currentMembership, nil
	}

	// Find the previous role to find its name.
	prevRole, err := s.getRoleByID(ctx, currentMembership.RoleID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get role by ID")
		return nil, err
	}

	// Find the new role for find its name.
	newRole, err := s.getRoleByID(ctx, namespaceMembership.RoleID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get role by ID")
		return nil, err
	}

	// If this namespace membership is an owner and this is a top-level group, verify it's not the only owner
	// to prevent the group from becoming orphaned
	if prevRole.Metadata.ID == models.OwnerRoleID.String() && newRole.Metadata.ID != models.OwnerRoleID.String() && currentMembership.Namespace.IsTopLevel() {
		if err = s.verifyNotOnlyOwner(ctx, currentMembership); err != nil {
			tracing.RecordError(span, err, "failed to verify this membership is not the only owner")
			return nil, err
		}
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer UpdateNamespaceMembership: %v", txErr)
		}
	}()

	updatedNamespaceMembership, err := s.dbClient.NamespaceMemberships.UpdateNamespaceMembership(txContext, namespaceMembership)
	if err != nil {
		tracing.RecordError(span, err, "failed to update namespace membership")
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &updatedNamespaceMembership.Namespace.Path,
			Action:        models.ActionUpdate,
			TargetType:    models.TargetNamespaceMembership,
			TargetID:      updatedNamespaceMembership.Metadata.ID,
			Payload: &models.ActivityEventUpdateNamespaceMembershipPayload{
				PrevRole: string(prevRole.Name),
				NewRole:  string(newRole.Name),
			},
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	return updatedNamespaceMembership, nil
}

func (s *service) DeleteNamespaceMembership(ctx context.Context, namespaceMembership *models.NamespaceMembership) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteNamespaceMembership")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	err := s.requirePermissionForNamespace(ctx, namespaceMembership.Namespace.Path, permissions.DeleteNamespaceMembershipPermission)
	if err != nil {
		tracing.RecordError(span, err, "namespace permission check failed")
		return err
	}

	// If this namespace membership is an owner and this is a top-level group, verify it's not the only owner
	// to prevent the group from becoming orphaned
	if namespaceMembership.RoleID == models.OwnerRoleID.String() && namespaceMembership.Namespace.IsTopLevel() {
		if err = s.verifyNotOnlyOwner(ctx, namespaceMembership); err != nil {
			tracing.RecordError(span, err, "failed to verify not the only owner")
			return err
		}
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer DeleteNamespaceMembership: %v", txErr)
		}
	}()

	if err = s.dbClient.NamespaceMemberships.DeleteNamespaceMembership(txContext, namespaceMembership); err != nil {
		tracing.RecordError(span, err, "failed to delete namespace membership")
		return err
	}

	eventTargetType, eventTargetID := getTargetTypeID(namespaceMembership)

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &namespaceMembership.Namespace.Path,
			Action:        models.ActionRemoveMembership,
			TargetType:    eventTargetType,
			TargetID:      eventTargetID,
			Payload: &models.ActivityEventRemoveNamespaceMembershipPayload{
				UserID:           namespaceMembership.UserID,
				ServiceAccountID: namespaceMembership.ServiceAccountID,
				TeamID:           namespaceMembership.TeamID,
			},
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return err
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) verifyNotOnlyOwner(ctx context.Context, namespaceMembership *models.NamespaceMembership) error {
	// Get all namespace memberships by group
	resp, err := s.dbClient.NamespaceMemberships.GetNamespaceMemberships(ctx, &db.GetNamespaceMembershipsInput{
		Filter: &db.NamespaceMembershipFilter{
			NamespacePaths: []string{namespaceMembership.Namespace.Path},
		},
	})
	if err != nil {
		return err
	}

	otherOwnerFound := false
	for _, m := range resp.NamespaceMemberships {
		if m.RoleID == models.OwnerRoleID.String() && m.Metadata.ID != namespaceMembership.Metadata.ID {
			otherOwnerFound = true
			break
		}
	}

	if !otherOwnerFound {
		return errors.New("namespace membership cannot be deleted because it's the only owner of group %s", namespaceMembership.Namespace.Path, errors.WithErrorCode(errors.EInvalid))
	}

	return nil
}

func (s *service) requirePermissionForNamespace(ctx context.Context, namespacePath string, perm permissions.Permission) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	return caller.RequirePermission(ctx, perm, auth.WithNamespacePath(namespacePath))
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

func getTargetTypeID(namespaceMembership *models.NamespaceMembership) (models.ActivityEventTargetType, string) {
	var eventTargetType models.ActivityEventTargetType
	var eventTargetID string
	if namespaceMembership.Namespace.GroupID != nil && *namespaceMembership.Namespace.GroupID != "" {
		eventTargetType = models.TargetGroup
		eventTargetID = *namespaceMembership.Namespace.GroupID
	} else {
		eventTargetType = models.TargetWorkspace
		eventTargetID = *namespaceMembership.Namespace.WorkspaceID
	}
	return eventTargetType, eventTargetID
}
