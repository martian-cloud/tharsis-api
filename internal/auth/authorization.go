package auth

//go:generate mockery --name Authorizer --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// Authorizer is used to authorize access to namespaces
type Authorizer interface {
	GetRootNamespaces(ctx context.Context) ([]models.MembershipNamespace, error)
	RequireAccess(ctx context.Context, perms []permissions.Permission, checks ...func(*constraints)) error
	RequireAccessToInheritableResource(ctx context.Context, resourceTypes []permissions.ResourceType, checks ...func(*constraints)) error
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
	rolePermissionCache      map[string][]permissions.Permission
	namespaceMembershipCache map[string]map[string]struct{}
	useCache                 bool
}

func newNamespaceMembershipAuthorizer(dbClient *db.Client, userID *string, serviceAccountID *string, useCache bool) *authorizer {
	return &authorizer{
		dbClient:                 dbClient,
		userID:                   userID,
		serviceAccountID:         serviceAccountID,
		useCache:                 useCache,
		rolePermissionCache:      map[string][]permissions.Permission{},
		namespaceMembershipCache: map[string]map[string]struct{}{},
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

func (a *authorizer) RequireAccess(ctx context.Context, perms []permissions.Permission, checks ...func(*constraints)) error {
	c := getConstraints(checks...)

	if (len(perms) == 0) || !a.hasConstraints(c, true) {
		return errMissingConstraints
	}

	for _, perm := range perms {
		permCopy := perm
		if c.groupID != nil {
			if err := a.requireAccessToGroup(ctx, *c.groupID, &permCopy); err != nil {
				return err
			}
		}
		if c.workspaceID != nil {
			if err := a.requireAccessToWorkspace(ctx, *c.workspaceID, &permCopy); err != nil {
				return err
			}
		}
		if len(c.namespacePaths) > 0 {
			if err := a.requireAccessToNamespaces(ctx, c.namespacePaths, &permCopy); err != nil {
				return err
			}
		}
	}

	return nil
}

func (a *authorizer) RequireAccessToInheritableResource(ctx context.Context, resourceTypes []permissions.ResourceType, checks ...func(*constraints)) error {
	c := getConstraints(checks...)

	// Must specify at least one resource type and constraint.
	if (len(resourceTypes) == 0) || !a.hasConstraints(c, false) {
		return errMissingConstraints
	}

	for _, rt := range resourceTypes {
		perm := &permissions.Permission{Action: permissions.ViewAction, ResourceType: rt}
		if c.groupID != nil {
			if err := a.requireAccessToInheritedGroupResource(ctx, *c.groupID, perm); err != nil {
				return err
			}
		}
		if len(c.namespacePaths) > 0 {
			if err := a.requireAccessToInheritedNamespaceResources(ctx, c.namespacePaths, perm); err != nil {
				return err
			}
		}
	}

	return nil
}

func (a *authorizer) requireAccessToGroup(ctx context.Context, groupID string, perm *permissions.Permission) error {
	// Check cache
	if a.checkCache(&cacheKey{groupID: &groupID}, perm) {
		return nil
	}

	group, err := a.dbClient.Groups.GetGroupByID(ctx, groupID)
	if err != nil {
		return err
	}

	if group == nil {
		return authorizationError(ctx, false)
	}

	return a.requireAccessToNamespace(ctx, group.FullPath, perm)
}

func (a *authorizer) requireAccessToWorkspace(ctx context.Context, workspaceID string, perm *permissions.Permission) error {
	// Check cache
	if a.checkCache(&cacheKey{workspaceID: &workspaceID}, perm) {
		return nil
	}

	ws, err := a.dbClient.Workspaces.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return err
	}

	if ws == nil {
		return authorizationError(ctx, false)
	}

	return a.requireAccessToNamespace(ctx, ws.FullPath, perm)
}

func (a *authorizer) requireAccessToNamespace(ctx context.Context, namespacePath string, perm *permissions.Permission) error {
	// Check cache
	if a.checkCache(&cacheKey{path: &namespacePath}, perm) {
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

	filteredMemberships := []models.NamespaceMembership{}
	seen := map[string]struct{}{}
	for _, nm := range resp.NamespaceMemberships {
		var id string
		switch {
		case nm.UserID != nil:
			id = *a.userID
		case nm.TeamID != nil:
			id = *nm.TeamID
		case nm.ServiceAccountID != nil:
			id = *nm.ServiceAccountID
		}

		if _, ok := seen[id]; ok {
			// Skip any parent memberships for same user, team or service account
			// since the lowest membership in the hierarchy should take precedence.
			continue
		}

		seen[id] = struct{}{}
		filteredMemberships = append(filteredMemberships, nm)
	}

	return a.requirePermission(ctx, filteredMemberships, perm)
}

func (a *authorizer) requireAccessToInheritedGroupResource(ctx context.Context, groupID string, perm *permissions.Permission) error {
	// Check cache
	if a.checkCache(&cacheKey{groupID: &groupID}, perm) {
		return nil
	}

	group, err := a.dbClient.Groups.GetGroupByID(ctx, groupID)
	if err != nil {
		return err
	}

	if group == nil {
		return authorizationError(ctx, false)
	}

	return a.requireAccessToInheritedNamespaceResource(ctx, group.FullPath, perm)
}

func (a *authorizer) requireAccessToInheritedNamespaceResource(ctx context.Context, namespace string, perm *permissions.Permission) error {
	// Check cache
	if a.checkCache(&cacheKey{path: &namespace}, perm) {
		return nil
	}

	namespaceParts := strings.Split(namespace, "/")
	resp, err := a.getNamespaceMemberships(ctx, &db.GetNamespaceMembershipsInput{
		Filter: &db.NamespaceMembershipFilter{
			NamespacePathPrefix: &namespaceParts[0],
		},
	})
	if err != nil {
		return err
	}

	if len(resp.NamespaceMemberships) == 0 {
		return authorizationError(ctx, false)
	}

	// Build a map of namespaces in descending order.
	expandedPaths := map[string]struct{}{}
	for i := len(namespaceParts); i > 0; i-- {
		expandedPaths[strings.Join(namespaceParts[0:i], "/")] = struct{}{}
	}

	memberships := []models.NamespaceMembership{}
	for _, nm := range resp.NamespaceMemberships {
		_, ok := expandedPaths[nm.Namespace.Path]
		if ok || strings.HasPrefix(nm.Namespace.Path, namespace+"/") {
			// Only add parent or child namespaces of requested namespace.
			memberships = append(memberships, nm)
		}
	}

	return a.requirePermission(ctx, memberships, perm)
}

func (a *authorizer) requireAccessToInheritedNamespaceResources(ctx context.Context, requiredNamespaces []string, perm *permissions.Permission) error {
	for _, ns := range requiredNamespaces {
		if err := a.requireAccessToInheritedNamespaceResource(ctx, ns, perm); err != nil {
			return err
		}
	}

	// Grant access
	return nil
}

func (a *authorizer) requireAccessToNamespaces(ctx context.Context, requiredNamespaces []string, perm *permissions.Permission) error {
	for _, ns := range requiredNamespaces {
		if err := a.requireAccessToNamespace(ctx, ns, perm); err != nil {
			return err
		}
	}

	// Grant access
	return nil
}

// checkCache returns true if the permission is found. It only looks for an exact
// match meaning a permission with View Action will not be automatically granted
// if a subject has a permission with Create, Update, Delete or Manage actions
// for that ResourceType.
func (a *authorizer) checkCache(key *cacheKey, perm *permissions.Permission) bool {
	if !a.useCache {
		return false
	}

	a.lock.RLock()
	defer a.lock.RUnlock()

	if key.path != nil {
		namespacePaths := expandNamespaceDescOrder(*key.path)

		for _, path := range namespacePaths {
			p := path
			if cachedPerms, ok := a.namespaceMembershipCache[cacheKey{path: &p}.getPathKey()]; ok {
				_, ok := cachedPerms[perm.String()]
				return ok
			}
		}
	} else if key.workspaceID != nil {
		if cachedPerms, ok := a.namespaceMembershipCache[key.getWorkspaceIDKey()]; ok {
			_, ok := cachedPerms[perm.String()]
			return ok
		}
	} else if key.groupID != nil {
		if cachedPerms, ok := a.namespaceMembershipCache[key.getGroupIDKey()]; ok {
			_, ok := cachedPerms[perm.String()]
			return ok
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

	return resp, nil
}

// addToCache adds specified perms to role and namespaceMembership caches.
func (a *authorizer) addToCache(membership *models.NamespaceMembership, perms []permissions.Permission) {
	if !a.useCache {
		// Cache isn't being used.
		return
	}

	a.lock.Lock()
	defer a.lock.Unlock()

	// Add role perms to cache.
	if _, ok := a.rolePermissionCache[membership.RoleID]; !ok {
		a.rolePermissionCache[membership.RoleID] = perms
	}

	// Add to membership cache.
	a.addMembershipToCache(cacheKey{path: &membership.Namespace.Path}.getPathKey(), perms)
	if membership.Namespace.WorkspaceID != nil {
		a.addMembershipToCache(cacheKey{workspaceID: membership.Namespace.WorkspaceID}.getWorkspaceIDKey(), perms)
	}
	if membership.Namespace.GroupID != nil {
		a.addMembershipToCache(cacheKey{groupID: membership.Namespace.GroupID}.getGroupIDKey(), perms)
	}
}

func (a *authorizer) addMembershipToCache(key string, perms []permissions.Permission) {
	if _, ok := a.namespaceMembershipCache[key]; !ok {
		permsMap := make(map[string]struct{}, len(perms))
		for _, p := range perms {
			permsMap[p.String()] = struct{}{}
		}
		a.namespaceMembershipCache[key] = permsMap
	}
}

func (a *authorizer) getPermissionsFromMembership(ctx context.Context, membership *models.NamespaceMembership) ([]permissions.Permission, error) {
	if a.useCache {
		a.lock.RLock()
		if perms, ok := a.rolePermissionCache[membership.RoleID]; ok {
			a.lock.RUnlock()
			return perms, nil
		}
		a.lock.RUnlock()
	}

	// Check if this a default role, in which case we can use the permissions from the map.
	if perms, ok := models.DefaultRoleID(membership.RoleID).Permissions(); ok {
		a.addToCache(membership, perms)
		return perms, nil
	}

	// Get the role by ID.
	role, err := a.dbClient.Roles.GetRoleByID(ctx, membership.RoleID)
	if err != nil {
		return nil, err
	}

	if role == nil {
		return nil, authorizationError(ctx, false)
	}

	perms := role.GetPermissions()

	// Add to membership and role cache.
	a.addToCache(membership, perms)

	return perms, nil
}

// requirePermission returns an error if the target permission can't be found within namespace memberships.
func (a *authorizer) requirePermission(
	ctx context.Context,
	memberships []models.NamespaceMembership,
	target *permissions.Permission,
) error {
	hasViewerAccess := false
	for _, membership := range memberships {
		membershipCopy := membership
		perms, err := a.getPermissionsFromMembership(ctx, &membershipCopy)
		if err != nil {
			return err
		}

		for _, p := range perms {
			if p.GTE(target) {
				return nil
			}

			// Determine if caller has at least viewer access for resource
			// if an authorizationError is to be returned.
			if p.ResourceType == target.ResourceType && p.Action.HasViewerAccess() {
				hasViewerAccess = true
			}
		}
	}

	return authorizationError(ctx, hasViewerAccess)
}

// hasConstraints returns true if at least one of the required constraints are specified.
func (*authorizer) hasConstraints(checks *constraints, checkWorkspace bool) bool {
	return checks.groupID != nil ||
		len(checks.namespacePaths) > 0 ||
		(checkWorkspace && checks.workspaceID != nil)
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
