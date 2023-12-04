package resolver

import (
	"context"
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/cli"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/gpgkey"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/group"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/job"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/moduleregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/namespacemembership"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/providermirror"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/providerregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/resourcelimit"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/role"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/runner"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/scim"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/serviceaccount"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/team"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/user"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/variable"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/vcs"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// Key type is used for attaching resolver state to the context
type key string

const (
	resolverStateKey key = "resolverState"
)

// State contains the services required by resolvers
type State struct {
	Config                     *config.Config
	GroupService               group.Service
	WorkspaceService           workspace.Service
	RunService                 run.Service
	JobService                 job.Service
	ManagedIdentityService     managedidentity.Service
	ServiceAccountService      serviceaccount.Service
	UserService                user.Service
	NamespaceMembershipService namespacemembership.Service
	VariableService            variable.Service
	Logger                     logger.Logger
	TeamService                team.Service
	ProviderRegistryService    providerregistry.Service
	ModuleRegistryService      moduleregistry.Service
	GPGKeyService              gpgkey.Service
	CliService                 cli.Service
	SCIMService                scim.Service
	VCSService                 vcs.Service
	ActivityService            activityevent.Service
	RoleService                role.Service
	RunnerService              runner.Service
	ResourceLimitService       resourcelimit.Service
	ProviderMirrorService      providermirror.Service
	MaintenanceModeService     maintenance.Service
}

// Attach is used to attach the resolver state to the context
func (r *State) Attach(ctx context.Context) context.Context {
	return context.WithValue(ctx, resolverStateKey, r)
}

func extract(ctx context.Context) *State {
	rs, ok := ctx.Value(resolverStateKey).(*State)
	if !ok {
		// Use panic here since this is not a recoverable error
		panic(fmt.Sprintf("unable to find %s resolver state on the request context", resolverStateKey))
	}

	return rs
}

func (k key) String() string {
	return fmt.Sprintf("gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/resolver %s", string(k))
}

func getGroupService(ctx context.Context) group.Service {
	return extract(ctx).GroupService
}

func getWorkspaceService(ctx context.Context) workspace.Service {
	return extract(ctx).WorkspaceService
}

func getRunService(ctx context.Context) run.Service {
	return extract(ctx).RunService
}

func getJobService(ctx context.Context) job.Service {
	return extract(ctx).JobService
}

func getManagedIdentityService(ctx context.Context) managedidentity.Service {
	return extract(ctx).ManagedIdentityService
}

func getSAService(ctx context.Context) serviceaccount.Service {
	return extract(ctx).ServiceAccountService
}

func getUserService(ctx context.Context) user.Service {
	return extract(ctx).UserService
}

func getNamespaceMembershipService(ctx context.Context) namespacemembership.Service {
	return extract(ctx).NamespaceMembershipService
}

func getVariableService(ctx context.Context) variable.Service {
	return extract(ctx).VariableService
}

func getProviderRegistryService(ctx context.Context) providerregistry.Service {
	return extract(ctx).ProviderRegistryService
}

func getModuleRegistryService(ctx context.Context) moduleregistry.Service {
	return extract(ctx).ModuleRegistryService
}

// nolint
func getLogger(ctx context.Context) logger.Logger {
	return extract(ctx).Logger
}

func getTeamService(ctx context.Context) team.Service {
	return extract(ctx).TeamService
}

func getGPGKeyService(ctx context.Context) gpgkey.Service {
	return extract(ctx).GPGKeyService
}

func getCLIService(ctx context.Context) cli.Service {
	return extract(ctx).CliService
}

func getSCIMService(ctx context.Context) scim.Service {
	return extract(ctx).SCIMService
}

func getVCSService(ctx context.Context) vcs.Service {
	return extract(ctx).VCSService
}

func getActivityService(ctx context.Context) activityevent.Service {
	return extract(ctx).ActivityService
}

func getRunnerService(ctx context.Context) runner.Service {
	return extract(ctx).RunnerService
}

func getConfig(ctx context.Context) *config.Config {
	return extract(ctx).Config
}

func getRoleService(ctx context.Context) role.Service {
	return extract(ctx).RoleService
}

func getResourceLimitService(ctx context.Context) resourcelimit.Service {
	return extract(ctx).ResourceLimitService
}

func getProviderMirrorService(ctx context.Context) providermirror.Service {
	return extract(ctx).ProviderMirrorService
}

func getMaintenanceModeService(ctx context.Context) maintenance.Service {
	return extract(ctx).MaintenanceModeService
}
