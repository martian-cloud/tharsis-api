package auth

//go:generate mockery --name Caller --inpackage --case underscore

import (
	"context"
	goerror "errors"
	"net/http"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// errMissingConstraints is the error returned when required constraints or permissions are missing.
var errMissingConstraints = goerror.New("missing required permissions or constraints")

// errInMaintenanceMode is the error returned when the system is in maintenance mode.
var errInMaintenanceMode = errors.New("System is currently in maintenance mode, only read operations are supported", errors.WithErrorCode(errors.EServiceUnavailable))

// Uses the context key pattern
type contextKey string

// contextKeyCaller accesses the caller object.
var contextKeyCaller = contextKey("caller")

// contextKeySubject accesses the subject string.
var contextKeySubject = contextKey("subject")

func (c contextKey) String() string {
	return "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth " + string(c)
}

// GetCaller returns a context's caller.  Return nil if no caller was found on the context.
func GetCaller(ctx context.Context) Caller {
	caller, ok := ctx.Value(contextKeyCaller).(Caller)
	if !ok {
		return nil
	}
	return caller
}

// GetSubject returns a context's subject.  Return nil if no subject was found on the context.
func GetSubject(ctx context.Context) *string {
	subject, ok := ctx.Value(contextKeySubject).(string)
	if !ok {
		return nil
	}
	return &subject
}

// permissionTypeHandler allows delegating checks based on the permission type.
type permissionTypeHandler func(ctx context.Context, perm *permissions.Permission, checks *constraints) error

// noopPermissionHandler handles any use-cases where the permission is automatically granted for a caller.
func noopPermissionHandler(_ context.Context, _ *permissions.Permission, _ *constraints) error {
	return nil
}

// constraints defines permission constraints that should be checked
// when a subject is being authorized to modify or view a Tharsis resource.
type constraints struct {
	workspaceID    *string
	groupID        *string
	planID         *string
	applyID        *string
	jobID          *string
	runID          *string
	teamID         *string
	userID         *string
	runnerID       *string
	namespacePaths []string // Group, workspace, namespace paths, etc.
}

// getConstraints returns a constraints struct.
func getConstraints(checks ...func(*constraints)) *constraints {
	constraint := &constraints{}
	for _, c := range checks {
		c(constraint)
	}

	return constraint
}

// WithWorkspaceID sets the WorkspaceID on constraints struct.
func WithWorkspaceID(id string) func(*constraints) {
	return func(c *constraints) {
		c.workspaceID = &id
	}
}

// WithNamespacePath sets the Namespace on constraints struct.
func WithNamespacePath(namespacePath string) func(*constraints) {
	return func(c *constraints) {
		c.namespacePaths = append(c.namespacePaths, namespacePath)
	}
}

// WithNamespacePaths sets the NamespacePaths on constraints struct.
func WithNamespacePaths(namespacePaths []string) func(*constraints) {
	return func(c *constraints) {
		c.namespacePaths = namespacePaths
	}
}

// WithPlanID sets the PlanID on constraints struct.
func WithPlanID(id string) func(*constraints) {
	return func(c *constraints) {
		c.planID = &id
	}
}

// WithApplyID sets the ApplyID on constraints struct.
func WithApplyID(id string) func(*constraints) {
	return func(c *constraints) {
		c.applyID = &id
	}
}

// WithJobID sets the JobID on constraints struct.
func WithJobID(id string) func(*constraints) {
	return func(c *constraints) {
		c.jobID = &id
	}
}

// WithRunID sets the RunID on constraints struct.
func WithRunID(id string) func(*constraints) {
	return func(c *constraints) {
		c.runID = &id
	}
}

// WithGroupID sets the GroupID on constraints struct.
func WithGroupID(id string) func(*constraints) {
	return func(c *constraints) {
		c.groupID = &id
	}
}

// WithUserID sets the UserID on constraints struct.
func WithUserID(id string) func(*constraints) {
	return func(c *constraints) {
		c.userID = &id
	}
}

// WithTeamID sets the TeamID on Constraints struct.
func WithTeamID(id string) func(*constraints) {
	return func(c *constraints) {
		c.teamID = &id
	}
}

// WithRunnerID sets the RunnerID on constraints struct.
func WithRunnerID(id string) func(*constraints) {
	return func(c *constraints) {
		c.runnerID = &id
	}
}

// SystemCaller is the caller subject for internal system calls
type SystemCaller struct{}

// GetSubject returns the subject identifier for this caller
func (s *SystemCaller) GetSubject() string {
	return "system"
}

// IsAdmin returns true if the caller is an admin
func (s *SystemCaller) IsAdmin() bool {
	// System caller is always an admin
	return true
}

// GetNamespaceAccessPolicy returns the namespace access policy for this caller
func (s *SystemCaller) GetNamespaceAccessPolicy(_ context.Context) (*NamespaceAccessPolicy, error) {
	return &NamespaceAccessPolicy{AllowAll: true}, nil
}

// RequirePermission will return an error if the caller doesn't have the specified permissions
func (s *SystemCaller) RequirePermission(_ context.Context, _ permissions.Permission, _ ...func(*constraints)) error {
	// Return nil because system caller is authorized to perform any action
	return nil
}

// RequireAccessToInheritableResource will return an error if the caller doesn't have access to the specified resource type
func (s *SystemCaller) RequireAccessToInheritableResource(_ context.Context, _ permissions.ResourceType, _ ...func(*constraints)) error {
	// Return nil because system caller is authorized to perform any action
	return nil
}

// UnauthorizedError returns the unauthorized error for this specific caller type
func (s *SystemCaller) UnauthorizedError(_ context.Context, _ bool) error {
	return errors.New(
		"system caller is not authorized to perform this action",
		errors.WithErrorCode(errors.EForbidden),
	)
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
	IsAdmin() bool
	GetNamespaceAccessPolicy(ctx context.Context) (*NamespaceAccessPolicy, error)
	RequirePermission(ctx context.Context, perms permissions.Permission, checks ...func(*constraints)) error
	RequireAccessToInheritableResource(ctx context.Context, resourceType permissions.ResourceType, checks ...func(*constraints)) error
	UnauthorizedError(ctx context.Context, hasViewerAccess bool) error
}

// WithCaller adds the caller to the context
func WithCaller(ctx context.Context, caller Caller) context.Context {
	return context.WithValue(ctx, contextKeyCaller, caller)
}

// WithSubject adds the subject string to the context
func WithSubject(ctx context.Context, subject string) context.Context {
	return context.WithValue(ctx, contextKeySubject, subject)
}

// AuthorizeCaller verifies that a caller has been authenticated and returns the caller
func AuthorizeCaller(ctx context.Context) (Caller, error) {
	caller, ok := ctx.Value(contextKeyCaller).(Caller)
	if !ok {
		return nil, errors.New("Authentication is required", errors.WithErrorCode(errors.EUnauthorized))
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
		return errors.New("Invalid caller type", errors.WithErrorCode(errors.EForbidden))
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
