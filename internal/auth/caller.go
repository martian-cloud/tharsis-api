package auth

//go:generate mockery --name Caller --inpackage --case underscore

import (
	"context"
	"net/http"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// Uses the context key pattern
type contextKey string

var (
	contextKeyCaller = contextKey("caller")
)

func (c contextKey) String() string {
	return "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth " + string(c)
}

// SystemCaller is the caller subject for internal system calls
type SystemCaller struct{}

// GetSubject returns the subject identifier for this caller
func (s *SystemCaller) GetSubject() string {
	return "system"
}

// GetNamespaceAccessPolicy returns the namespace access policy for this caller
func (s *SystemCaller) GetNamespaceAccessPolicy(ctx context.Context) (*NamespaceAccessPolicy, error) {
	return &NamespaceAccessPolicy{AllowAll: true}, nil
}

// RequireAccessToNamespace will return an error if the caller doesn't have the specified access level
func (s *SystemCaller) RequireAccessToNamespace(ctx context.Context, namespacePath string, accessLevel models.Role) error {
	// Return nil because system caller is authorized to perform any action
	return nil
}

// RequireViewerAccessToGroups will return an error if the caller doesn't have viewer access to all the specified groups
func (s *SystemCaller) RequireViewerAccessToGroups(ctx context.Context, groups []models.Group) error {
	// Return nil because system caller is authorized to perform any action
	return nil
}

// RequireViewerAccessToWorkspaces will return an error if the caller doesn't have viewer access on the specified workspace
func (s *SystemCaller) RequireViewerAccessToWorkspaces(ctx context.Context, workspaces []models.Workspace) error {
	// Return nil because system caller is authorized to perform any action
	return nil
}

// RequireViewerAccessToNamespaces will return an error if the caller doesn't have viewer access to the specified list of namespaces
func (s *SystemCaller) RequireViewerAccessToNamespaces(ctx context.Context, namespaces []string) error {
	// Return nil because system caller is authorized to perform any action
	return nil
}

// RequireAccessToGroup will return an error if the caller doesn't have the required access level on the specified group
func (s *SystemCaller) RequireAccessToGroup(ctx context.Context, groupID string, accessLevel models.Role) error {
	// Return nil because system caller is authorized to perform any action
	return nil
}

// RequireAccessToWorkspace will return an error if the caller doesn't have the required access level on the specified workspace
func (s *SystemCaller) RequireAccessToWorkspace(ctx context.Context, workspaceID string, accessLevel models.Role) error {
	// Return nil because system caller is authorized to perform any action
	return nil
}

// RequireAccessToInheritedGroupResource will return an error if the caller doesn't have viewer access on any namespace within the namespace hierarchy
func (s *SystemCaller) RequireAccessToInheritedGroupResource(ctx context.Context, groupID string) error {
	// Return nil because system caller is authorized to perform any action
	return nil
}

// RequireAccessToInheritedNamespaceResource will return an error if the caller doesn't have viewer access on any namespace within the namespace hierarchy
func (s *SystemCaller) RequireAccessToInheritedNamespaceResource(ctx context.Context, namespace string) error {
	// Return nil because system caller is authorized to perform any action
	return nil
}

// RequireRunWriteAccess will return an error if the caller doesn't have permission to update run state
func (s *SystemCaller) RequireRunWriteAccess(ctx context.Context, runID string) error {
	// Return nil because system caller is authorized to perform any action
	return nil
}

// RequirePlanWriteAccess will return an error if the caller doesn't have permission to update plan state
func (s *SystemCaller) RequirePlanWriteAccess(ctx context.Context, planID string) error {
	// Return nil because system caller is authorized to perform any action
	return nil
}

// RequireApplyWriteAccess will return an error if the caller doesn't have permission to update apply state
func (s *SystemCaller) RequireApplyWriteAccess(ctx context.Context, applyID string) error {
	// Return nil because system caller is authorized to perform any action
	return nil
}

// RequireJobWriteAccess will return an error if the caller doesn't have permission to update the state of the specified job
func (s *SystemCaller) RequireJobWriteAccess(ctx context.Context, jobID string) error {
	// Return nil because system caller is authorized to perform any action
	return nil
}

// RequireTeamCreateAccess will return an error if the caller does not have permission for the specified access on the specified team.
func (s *SystemCaller) RequireTeamCreateAccess(ctx context.Context) error {
	// Return nil because system caller is authorized to perform any action
	return nil
}

// RequireTeamUpdateAccess will return an error if the caller does not have permission for the specified access on the specified team.
func (s *SystemCaller) RequireTeamUpdateAccess(ctx context.Context, teamID string) error {
	// Return nil because system caller is authorized to perform any action
	return nil
}

// RequireTeamDeleteAccess will return an error if the caller does not have permission for the specified access on the specified team.
func (s *SystemCaller) RequireTeamDeleteAccess(ctx context.Context, teamID string) error {
	// Return nil because system caller is authorized to perform any action
	return nil
}

// RequireUserCreateAccess will return an error if the specified caller is not allowed to create users.
func (s *SystemCaller) RequireUserCreateAccess(ctx context.Context) error {
	// Return nil because system caller is authorized to perform any action
	return nil
}

// RequireUserUpdateAccess will return an error if the specified caller is not allowed to update a user.
func (s *SystemCaller) RequireUserUpdateAccess(ctx context.Context, userID string) error {
	// Return nil because system caller is authorized to perform any action
	return nil
}

// RequireUserDeleteAccess will return an error if the specified caller is not allowed to delete a user.
func (s *SystemCaller) RequireUserDeleteAccess(ctx context.Context, userID string) error {
	// Return nil because system caller is authorized to perform any action
	return nil
}

// NamespaceAccessPolicy specifies the namespaces that a caller has access to
type NamespaceAccessPolicy struct {
	// RootNamespaceIDs restricts the caller to the specified root namespaces
	RootNamespaceIDs []string
	// AllowAll indicates that the caller has access to all namespaces
	AllowAll bool
}

// Caller represents a subject performing an API request
type Caller interface {
	GetSubject() string
	GetNamespaceAccessPolicy(ctx context.Context) (*NamespaceAccessPolicy, error)
	RequireAccessToNamespace(ctx context.Context, namespacePath string, accessLevel models.Role) error
	RequireViewerAccessToGroups(ctx context.Context, groups []models.Group) error
	RequireViewerAccessToWorkspaces(ctx context.Context, workspaces []models.Workspace) error
	RequireViewerAccessToNamespaces(ctx context.Context, namespaces []string) error
	RequireAccessToGroup(ctx context.Context, groupID string, accessLevel models.Role) error
	RequireAccessToWorkspace(ctx context.Context, workspaceID string, accessLevel models.Role) error
	RequireAccessToInheritedGroupResource(ctx context.Context, groupID string) error
	RequireAccessToInheritedNamespaceResource(ctx context.Context, namespace string) error
	RequireRunWriteAccess(ctx context.Context, runID string) error
	RequirePlanWriteAccess(ctx context.Context, planID string) error
	RequireApplyWriteAccess(ctx context.Context, applyID string) error
	RequireJobWriteAccess(ctx context.Context, jobID string) error
	RequireTeamCreateAccess(ctx context.Context) error
	RequireTeamUpdateAccess(ctx context.Context, teamID string) error
	RequireTeamDeleteAccess(ctx context.Context, teamID string) error
	RequireUserCreateAccess(ctx context.Context) error
	RequireUserUpdateAccess(ctx context.Context, userID string) error
	RequireUserDeleteAccess(ctx context.Context, userID string) error
}

// WithCaller adds the caller to the context
func WithCaller(ctx context.Context, caller Caller) context.Context {
	return context.WithValue(ctx, contextKeyCaller, caller)
}

// AuthorizeCaller verifies that a caller has been authenticated and returns the caller
func AuthorizeCaller(ctx context.Context) (Caller, error) {
	caller, ok := ctx.Value(contextKeyCaller).(Caller)
	if !ok {
		return nil, errors.NewError(errors.EUnauthorized, "Authentication is required")
	}

	return caller, nil
}

// HandleCaller will invoke the provided callback based on the type of caller
func HandleCaller(
	ctx context.Context,
	userHandler func(ctx context.Context, caller *UserCaller) error,
	serviceAccountHandler func(ctx context.Context, caller *ServiceAccountCaller) error,
) error {
	caller, err := AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	switch c := caller.(type) {
	case *UserCaller:
		return userHandler(ctx, c)
	case *ServiceAccountCaller:
		return serviceAccountHandler(ctx, c)
	default:
		return errors.NewError(errors.EForbidden, "Invalid caller type")
	}
}

// FindToken returns the bearer token from an HTTP request
func FindToken(r *http.Request) string {
	// Get token from authorization header.
	bearer := r.Header.Get("Authorization")
	if len(bearer) > 7 && strings.ToUpper(bearer[0:6]) == "BEARER" {
		return bearer[7:]
	}

	return ""
}
