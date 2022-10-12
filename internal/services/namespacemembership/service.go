package namespacemembership

//go:generate mockery --name Service --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// CreateNamespaceMembershipInput is the input for creating a new namespace membership
type CreateNamespaceMembershipInput struct {
	User           *models.User
	ServiceAccount *models.ServiceAccount
	Team           *models.Team
	NamespacePath  string
	Role           models.Role
}

// GetNamespaceMembershipsForSubjectInput is the input for querying a list of namespace memberships
type GetNamespaceMembershipsForSubjectInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.NamespaceMembershipSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *db.PaginationOptions
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
	CreateNamespaceMembership(ctx context.Context, input *CreateNamespaceMembershipInput) (*models.NamespaceMembership, error)
	UpdateNamespaceMembership(ctx context.Context, namespaceMembership *models.NamespaceMembership) (*models.NamespaceMembership, error)
	DeleteNamespaceMembership(ctx context.Context, namespaceMembership *models.NamespaceMembership) error
}

type service struct {
	logger   logger.Logger
	dbClient *db.Client
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
) Service {
	return &service{
		logger:   logger,
		dbClient: dbClient,
	}
}

func (s *service) GetNamespaceMembershipsForNamespace(ctx context.Context, namespacePath string) ([]models.NamespaceMembership, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err = caller.RequireAccessToNamespace(ctx, namespacePath, models.ViewerRole); err != nil {
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
	input *GetNamespaceMembershipsForSubjectInput) (*db.NamespaceMembershipResult, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// Exactly one of these should take effect.
	switch {
	case input.UserID != nil:
		userCaller, ok := caller.(*auth.UserCaller)
		if !ok || (!userCaller.User.Admin && userCaller.User.Metadata.ID != *input.UserID) {
			return nil, errors.NewError(errors.EForbidden,
				fmt.Sprintf("User %s is not authorized to query namespace memberships for %s", userCaller.User.Username, *input.UserID))
		}
	case input.ServiceAccount != nil:
		// Verify caller has access to the group this service account is in
		if err := caller.RequireAccessToGroup(ctx, input.ServiceAccount.GroupID, models.ViewerRole); err != nil {
			return nil, err
		}
	default:
		return nil, errors.NewError(errors.EInvalid, "input is missing required fields")
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
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	namespaceMembership, err := s.dbClient.NamespaceMemberships.GetNamespaceMembershipByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if namespaceMembership == nil {
		return nil, errors.NewError(errors.ENotFound, fmt.Sprintf("namespace membership with id %s not found", id))
	}

	if err := caller.RequireAccessToNamespace(ctx, namespaceMembership.Namespace.Path, models.ViewerRole); err != nil {
		return nil, err
	}

	return namespaceMembership, nil
}

func (s *service) CreateNamespaceMembership(ctx context.Context,
	input *CreateNamespaceMembershipInput) (*models.NamespaceMembership, error) {
	if err := s.requireOwnerAccessToNamespace(ctx, input.NamespacePath); err != nil {
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
		return nil, errors.NewError(errors.EInvalid, "Exactly one of User, ServiceAccount, team field must be defined")
	}

	// If this is a service account, we need to verify that it's being added to the group that it is associated with
	// or a nested group
	if input.ServiceAccount != nil {
		// Remove service account name from resource path
		parts := strings.Split(input.ServiceAccount.ResourcePath, "/")
		serviceAccountNamespace := strings.Join(parts[:len(parts)-1], "/")

		if serviceAccountNamespace != input.NamespacePath && !strings.HasPrefix(input.NamespacePath, serviceAccountNamespace+"/") {
			return nil, errors.NewError(
				errors.EInvalid,
				fmt.Sprintf(
					"Service account cannot be added as a member to group %s because it doesn't exist in the group or a parent group",
					input.NamespacePath,
				),
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

	namespaceMembership, err := s.dbClient.NamespaceMemberships.CreateNamespaceMembership(ctx, &db.CreateNamespaceMembershipInput{
		NamespacePath:    input.NamespacePath,
		Role:             input.Role,
		UserID:           userID,
		ServiceAccountID: serviceAccountID,
		TeamID:           teamID,
	})
	if err != nil {
		return nil, err
	}

	return namespaceMembership, nil
}

func (s *service) UpdateNamespaceMembership(ctx context.Context,
	namespaceMembership *models.NamespaceMembership) (*models.NamespaceMembership, error) {
	if err := s.requireOwnerAccessToNamespace(ctx, namespaceMembership.Namespace.Path); err != nil {
		return nil, err
	}

	// Get current state of namespace membership
	currentNamespaceMembership, err := s.dbClient.NamespaceMemberships.GetNamespaceMembershipByID(ctx, namespaceMembership.Metadata.ID)
	if err != nil {
		return nil, err
	}

	if currentNamespaceMembership == nil {
		return nil, errors.NewError(errors.ENotFound, fmt.Sprintf("namespace membership with ID %s not found", namespaceMembership.Metadata.ID))
	}

	// If this namespace membership is an owner and this is a top-level group, verify it's not the only owner
	// to prevent the group from becoming orphaned
	if currentNamespaceMembership.Role == models.OwnerRole && namespaceMembership.Role != models.OwnerRole && currentNamespaceMembership.Namespace.IsTopLevel() {
		if err = s.verifyNotOnlyOwner(ctx, currentNamespaceMembership); err != nil {
			return nil, err
		}
	}

	updatedNamespaceMembership, err := s.dbClient.NamespaceMemberships.UpdateNamespaceMembership(ctx, namespaceMembership)
	if err != nil {
		return nil, err
	}

	return updatedNamespaceMembership, nil
}

func (s *service) DeleteNamespaceMembership(ctx context.Context, namespaceMembership *models.NamespaceMembership) error {
	if err := s.requireOwnerAccessToNamespace(ctx, namespaceMembership.Namespace.Path); err != nil {
		return err
	}

	// If this namespace membership is an owner and this is a top-level group, verify it's not the only owner
	// to prevent the group from becoming orphaned
	if namespaceMembership.Role == models.OwnerRole && namespaceMembership.Namespace.IsTopLevel() {
		if err := s.verifyNotOnlyOwner(ctx, namespaceMembership); err != nil {
			return err
		}
	}

	return s.dbClient.NamespaceMemberships.DeleteNamespaceMembership(ctx, namespaceMembership)
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
		if m.Role == models.OwnerRole && m.Metadata.ID != namespaceMembership.Metadata.ID {
			otherOwnerFound = true
			break
		}
	}

	if !otherOwnerFound {
		return errors.NewError(errors.EInvalid, fmt.Sprintf("namespace membership cannot be deleted because it's the only owner of group %s", namespaceMembership.Namespace.Path))
	}

	return nil
}

func (s *service) requireOwnerAccessToNamespace(ctx context.Context, namespacePath string) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	parts := strings.Split(namespacePath, "/")
	namespacePaths := []string{}

	// Namespaces are added in descending order
	for i := len(parts); i > 0; i-- {
		namespacePaths = append(namespacePaths, strings.Join(parts[0:i], "/"))
	}

	for i, path := range namespacePaths {
		err := caller.RequireAccessToNamespace(ctx, path, models.OwnerRole)
		if err != nil {
			// Only return error if all namespaces have been checked
			if i == (len(namespacePaths) - 1) {
				return err
			}
		} else {
			// Break because caller has owner access
			break
		}
	}
	return nil
}
