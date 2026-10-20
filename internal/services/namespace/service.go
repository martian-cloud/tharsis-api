// Package namespace provides the namespace listing service.
package namespace

//go:generate go tool mockery --name Service --inpackage --case underscore

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// GetNamespacesInput is the input for GetNamespaces.
type GetNamespacesInput struct {
	// PaginationOptions supports cursor-based pagination.
	PaginationOptions *pagination.Options
	// ParentGroupID filters to direct children of this group (internal UUID).
	ParentGroupID *string
	// IncludeNested widens ParentGroupID to every descendant of the group rather than only direct children.
	IncludeNested bool
	// Search matches anywhere in the path; with IncludeNested it only matches the part below the parent group (case-insensitive contains).
	Search *string
	// TopLevel restricts results to the top of the caller's namespace hierarchy.
	TopLevel bool
	// Sort specifies the sort order (default FULL_PATH_ASC).
	Sort *db.NamespaceSortableField
}

// Service provides namespace listing functionality.
type Service interface {
	// GetNamespace returns the group or workspace at the given full path.
	GetNamespace(ctx context.Context, fullPath string) (*db.Namespace, error)
	// GetNamespaces returns a paginated list of groups and workspaces as namespaces.
	GetNamespaces(ctx context.Context, input *GetNamespacesInput) (*db.NamespacesResult, error)
}

type service struct {
	dbClient *db.Client
}

// New creates an instance of Service.
func New(dbClient *db.Client) Service {
	return &service{dbClient: dbClient}
}

func (s *service) GetNamespace(ctx context.Context, fullPath string) (*db.Namespace, error) {
	ctx, span := tracer.Start(ctx, "svc.GetNamespace")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	result, err := s.dbClient.Namespaces.GetNamespace(ctx, fullPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get namespace", errors.WithSpan(span))
	}

	if result == nil {
		return nil, errors.New("namespace %s not found", fullPath, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	if result.Group != nil {
		if err = caller.RequirePermission(ctx, models.ViewGroupPermission, auth.WithNamespacePath(fullPath)); err != nil {
			return nil, err
		}
		return result, nil
	}

	if err = caller.RequirePermission(ctx, models.ViewWorkspacePermission, auth.WithNamespacePath(fullPath)); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *service) GetNamespaces(ctx context.Context, input *GetNamespacesInput) (*db.NamespacesResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetNamespaces")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	dbFilter := &db.NamespaceFilter{
		Search:   input.Search,
		TopLevel: input.TopLevel,
	}

	if input.ParentGroupID != nil {
		err = caller.RequirePermission(ctx, models.ViewGroupPermission, auth.WithGroupID(*input.ParentGroupID))
		if err != nil {
			return nil, err
		}

		parent, gErr := s.dbClient.Groups.GetGroupByID(ctx, *input.ParentGroupID)
		if gErr != nil {
			return nil, errors.Wrap(gErr, "failed to get parent group", errors.WithSpan(span))
		}
		if parent == nil {
			return nil, errors.New("group with id %s not found", *input.ParentGroupID, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
		}
		dbFilter.ParentPath = &parent.FullPath
		dbFilter.IncludeNested = input.IncludeNested
	} else if !caller.IsAdminModeActivated(ctx) {
		rootNamespaces, rErr := caller.GetRootNamespaceMemberships(ctx)
		if rErr != nil {
			return nil, errors.Wrap(rErr, "failed to get root namespace memberships", errors.WithSpan(span))
		}
		// A nil slice means "no filter" to the db layer, so a non-admin caller that has no
		// memberships must be given an empty non-nil slice, which matches nothing
		if rootNamespaces == nil {
			rootNamespaces = []models.MembershipNamespace{}
		}
		dbFilter.RootNamespaceMemberships = rootNamespaces
	}

	result, err := s.dbClient.Namespaces.GetNamespaces(ctx, &db.GetNamespacesInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter:            dbFilter,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get namespaces from db", errors.WithSpan(span))
	}

	return result, nil
}
