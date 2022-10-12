package auth

//go:generate mockery --name Authorizer --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// Authorizer is used to authorize access to namespaces
type Authorizer interface {
	GetRootNamespaces(ctx context.Context) ([]models.MembershipNamespace, error)
	RequireAccessToGroup(ctx context.Context, groupID string, accessLevel models.Role) error
	RequireAccessToWorkspace(ctx context.Context, workspaceID string, accessLevel models.Role) error
	RequireAccessToNamespace(ctx context.Context, namespacePath string, accessLevel models.Role) error
	RequireViewerAccessToGroups(ctx context.Context, groups []models.Group) error
	RequireViewerAccessToWorkspaces(ctx context.Context, workspaces []models.Workspace) error
	RequireViewerAccessToNamespaces(ctx context.Context, requiredNamespaces []string) error
	RequireAccessToInheritedGroupResource(ctx context.Context, groupID string) error
	RequireAccessToInheritedNamespaceResource(ctx context.Context, namespace string) error
}

const (
	resourceNotFoundErrorMsg = "Resource not found"
)

type cacheKey struct {
	path        *string
	workspaceID *string
	groupID     *string
}

func (c cacheKey) getPathKey() string {
	return fmt.Sprintf("path::%s", *c.path)
}

func (c cacheKey) getWorkspaceIDKey() string {
	return fmt.Sprintf("workspace_id::%s", *c.workspaceID)
}

func (c cacheKey) getGroupIDKey() string {
	return fmt.Sprintf("group_id::%s", *c.groupID)
}

type authorizer struct {
	lock                     sync.RWMutex
	dbClient                 *db.Client
	userID                   *string
	serviceAccountID         *string
	namespaceMembershipCache map[string]models.Role
	useCache                 bool
}

func newNamespaceMembershipAuthorizer(dbClient *db.Client, userID *string, serviceAccountID *string, useCache bool) *authorizer {
	return &authorizer{
		dbClient:                 dbClient,
		userID:                   userID,
		serviceAccountID:         serviceAccountID,
		useCache:                 useCache,
		namespaceMembershipCache: map[string]models.Role{},
	}
}

func (a *authorizer) GetRootNamespaces(ctx context.Context) ([]models.MembershipNamespace, error) {
	sortBy := db.NamespaceMembershipSortableFieldNamespacePathAsc
	resp, err := a.getNamespaceMemberships(ctx, &db.GetNamespaceMembershipsInput{
		Sort: &sortBy,
	})
	if err != nil {
		return nil, err
	}

	namespaces := map[string]models.MembershipNamespace{}
	for _, m := range resp.NamespaceMemberships {
		pathParts := strings.Split(m.Namespace.Path, "/")
		// Subtract 1 since we will always add root namespaces
		lastIndex := len(pathParts) - 1
		// Check each parent path to determine the top level namespace
		found := false
		if lastIndex > 0 {
			// This check excludes the last path since we're only checking parent paths
			for _, part := range pathParts[:lastIndex] {
				if _, ok := namespaces[part]; ok {
					// part is already in namespace map
					found = true
					break
				}
			}
		}
		// Add namespace if it hasn't already been added
		if !found {
			namespaces[pathParts[lastIndex]] = m.Namespace
		}
	}

	rootNamespaces := []models.MembershipNamespace{}

	for _, v := range namespaces {
		rootNamespaces = append(rootNamespaces, v)
	}

	return rootNamespaces, nil
}

func (a *authorizer) RequireAccessToGroup(ctx context.Context, groupID string, accessLevel models.Role) error {
	// Check cache
	if a.checkCache(&cacheKey{groupID: &groupID}, accessLevel) {
		return nil
	}

	group, err := a.dbClient.Groups.GetGroupByID(ctx, groupID)
	if err != nil {
		return err
	}

	if group == nil {
		return authorizationError(ctx, false)
	}

	return a.RequireAccessToNamespace(ctx, group.FullPath, accessLevel)
}

func (a *authorizer) RequireAccessToWorkspace(ctx context.Context, workspaceID string, accessLevel models.Role) error {
	// Check cache
	if a.checkCache(&cacheKey{workspaceID: &workspaceID}, accessLevel) {
		return nil
	}

	ws, err := a.dbClient.Workspaces.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return err
	}

	if ws == nil {
		return authorizationError(ctx, false)
	}

	return a.RequireAccessToNamespace(ctx, ws.FullPath, accessLevel)
}

func (a *authorizer) RequireAccessToNamespace(ctx context.Context, namespacePath string, accessLevel models.Role) error {
	// Check cache
	if a.checkCache(&cacheKey{path: &namespacePath}, accessLevel) {
		return nil
	}

	// Descending sort is used so we can traverse the namespace hierarchy from the bottom up
	// Don't limit the query to one result, because team member relationships can result in many rows.
	sortBy := db.NamespaceMembershipSortableFieldNamespacePathDesc
	resp, err := a.getNamespaceMemberships(ctx, &db.GetNamespaceMembershipsInput{
		Sort: &sortBy,
		Filter: &db.NamespaceMembershipFilter{
			NamespacePaths: expandNamespaceDescOrder(namespacePath),
		},
	})
	if err != nil {
		return err
	}

	for _, m := range resp.NamespaceMemberships {
		if m.Role.GTE(accessLevel) {
			// Grant access
			return nil
		}
	}

	return authorizationError(ctx, len(resp.NamespaceMemberships) > 0)
}

func (a *authorizer) RequireViewerAccessToGroups(ctx context.Context, groups []models.Group) error {
	namespaces := []string{}
	for _, group := range groups {
		namespaces = append(namespaces, group.FullPath)
	}

	return a.RequireViewerAccessToNamespaces(ctx, namespaces)
}

func (a *authorizer) RequireViewerAccessToWorkspaces(ctx context.Context, workspaces []models.Workspace) error {
	namespaces := []string{}
	for _, ws := range workspaces {
		namespaces = append(namespaces, ws.FullPath)
	}

	return a.RequireViewerAccessToNamespaces(ctx, namespaces)
}

func (a *authorizer) RequireAccessToInheritedGroupResource(ctx context.Context, groupID string) error {
	// Check cache
	if a.checkCache(&cacheKey{groupID: &groupID}, models.ViewerRole) {
		return nil
	}

	group, err := a.dbClient.Groups.GetGroupByID(ctx, groupID)
	if err != nil {
		return err
	}

	if group == nil {
		return authorizationError(ctx, false)
	}

	return a.RequireAccessToInheritedNamespaceResource(ctx, group.FullPath)
}

func (a *authorizer) RequireAccessToInheritedNamespaceResource(ctx context.Context, namespace string) error {
	// Check cache
	if a.checkCache(&cacheKey{path: &namespace}, models.ViewerRole) {
		return nil
	}

	namespaceParts := strings.Split(namespace, "/")

	resp, err := a.getNamespaceMemberships(ctx, &db.GetNamespaceMembershipsInput{
		Filter: &db.NamespaceMembershipFilter{
			// Filter by namespace prefix
			NamespacePathPrefix: &namespaceParts[0],
		},
	})
	if err != nil {
		return err
	}

	if len(resp.NamespaceMemberships) == 0 {
		return authorizationError(ctx, false)
	}

	return nil
}

func (a *authorizer) RequireViewerAccessToNamespaces(ctx context.Context, requiredNamespaces []string) error {
	if a.useCache {
		cacheMiss := false
		for _, ns := range requiredNamespaces {
			path := ns
			if !a.checkCache(&cacheKey{path: &path}, models.ViewerRole) {
				cacheMiss = true
				break
			}
		}
		if !cacheMiss {
			// grant access because all required access levels were in the cache
			return nil
		}
	}

	rootNamespaces, err := a.GetRootNamespaces(ctx)
	if err != nil {
		return err
	}

	rootNamespaceMap := map[string]bool{}
	for _, ns := range rootNamespaces {
		rootNamespaceMap[ns.Path] = true
	}

	for _, ns := range requiredNamespaces {
		paths := expandNamespaceDescOrder(ns)
		found := false
		// If any path of the namespace path is found in the map then the user has viewer access
		for _, path := range paths {
			if _, ok := rootNamespaceMap[path]; ok {
				found = true
				break
			}
		}

		if !found {
			return authorizationError(ctx, false)
		}
	}

	// Grant access
	return nil
}

func (a *authorizer) checkCache(key *cacheKey, accessLevel models.Role) bool {
	if !a.useCache {
		return false
	}

	a.lock.RLock()
	defer a.lock.RUnlock()

	if key.path != nil {
		namespacePaths := expandNamespaceDescOrder(*key.path)

		for _, path := range namespacePaths {
			p := path
			level, ok := a.namespaceMembershipCache[cacheKey{path: &p}.getPathKey()]
			if ok {
				// Check first role found
				return level.GTE(accessLevel)
			}
		}
	} else if key.workspaceID != nil {
		level, ok := a.namespaceMembershipCache[key.getWorkspaceIDKey()]
		if ok && level.GTE(accessLevel) {
			return true
		}
	} else if key.groupID != nil {
		level, ok := a.namespaceMembershipCache[key.getGroupIDKey()]
		if ok && level.GTE(accessLevel) {
			return true
		}
	}

	return false
}

func (a *authorizer) getNamespaceMemberships(ctx context.Context,
	input *db.GetNamespaceMembershipsInput) (*db.NamespaceMembershipResult, error) {
	if input.Filter == nil {
		input.Filter = &db.NamespaceMembershipFilter{}
	}

	input.Filter.UserID = a.userID
	input.Filter.ServiceAccountID = a.serviceAccountID

	resp, err := a.dbClient.NamespaceMemberships.GetNamespaceMemberships(ctx, input)
	if err != nil {
		return nil, err
	}

	a.lock.Lock()
	defer a.lock.Unlock()

	for _, membership := range resp.NamespaceMemberships {
		a.addMembershipToCache(cacheKey{path: &membership.Namespace.Path}.getPathKey(), membership.Role)
		if membership.Namespace.WorkspaceID != nil {
			a.addMembershipToCache(cacheKey{workspaceID: membership.Namespace.WorkspaceID}.getWorkspaceIDKey(), membership.Role)
		}
		if membership.Namespace.GroupID != nil {
			a.addMembershipToCache(cacheKey{groupID: membership.Namespace.GroupID}.getGroupIDKey(), membership.Role)
		}
	}

	return resp, nil
}

func (a *authorizer) addMembershipToCache(key string, role models.Role) {
	curRole, ok := a.namespaceMembershipCache[key]
	// Only add role to cache if it's GTE to the existing role that is cached
	if !ok || !curRole.GTE(role) {
		a.namespaceMembershipCache[key] = role
	}
}

func authorizationError(ctx context.Context, hasViewerAccessLevel bool) error {
	caller, err := AuthorizeCaller(ctx)
	if err != nil {
		return err
	}
	// If subject has at least viewer permissions then return 403, if not, return 404
	if hasViewerAccessLevel {
		return errors.NewError(errors.EForbidden, fmt.Sprintf("%s is not authorized to perform the requested operation", caller.GetSubject()))
	}
	return errors.NewError(errors.ENotFound, resourceNotFoundErrorMsg)
}

func expandNamespaceDescOrder(path string) []string {
	parts := strings.Split(path, "/")
	namespacePaths := []string{}

	// Namespaces need to be returned in descending order
	for i := len(parts); i > 0; i-- {
		namespacePaths = append(namespacePaths, strings.Join(parts[0:i], "/"))
	}
	return namespacePaths
}
