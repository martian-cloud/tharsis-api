package run

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/registry"
	runvariables "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/variables"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	modeltypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// ResolvedModule is the output of ResolveModule: the parsed registry source (nil for a
// config-version run or a non-registry remote source) along with the resolved exact semantic
// version and digest (both nil when there is no registry module).
type ResolvedModule struct {
	Source  registry.ModuleRegistrySource
	Version *string
	Digest  []byte
}

// ResolveModule parses a registry module source and resolves its exact semantic version and digest,
// verifying that the caller is authorized to use a private module. It performs registry/network
// I/O, so callers run it in their Prepare phase before the transaction is opened. The resolved
// values are then passed to Create.
//
// It returns a zero-value ResolvedModule when moduleSource is nil (a configuration-version run) or
// when the source is a remote source that doesn't use the registry protocol.
func ResolveModule(
	ctx context.Context,
	dbClient *db.Client,
	moduleResolver registry.ModuleResolver,
	workspaceID string,
	moduleSource *string,
	wantVersion *string,
	includeModulePrereleases bool,
	variables []runvariables.Variable,
) (*ResolvedModule, error) {
	resolved := &ResolvedModule{}
	if moduleSource == nil {
		return resolved, nil
	}

	// Collect the environment variables (used to resolve module-registry tokens).
	runEnvVars := []runvariables.Variable{}
	for _, variable := range variables {
		if variable.Category == models.EnvironmentVariableCategory {
			runEnvVars = append(runEnvVars, variable)
		}
	}
	tokenGetter := runvariables.ModuleRegistryToken(runEnvVars)

	// Normalize the module version: strip a leading 'v' followed by a digit so that "v1.0.0"
	// resolves identically to "1.0.0". Constraint expressions (e.g., ">= 1.0.0", ">= v1.0.0,
	// < v2.0.0") are left unchanged — the hashicorp/go-version library handles any embedded 'v'
	// prefixes natively.
	normalizedModuleVersion := wantVersion
	if normalizedModuleVersion != nil && len(*normalizedModuleVersion) > 1 &&
		(*normalizedModuleVersion)[0] == 'v' && (*normalizedModuleVersion)[1] >= '0' && (*normalizedModuleVersion)[1] <= '9' {
		stripped := (*normalizedModuleVersion)[1:]
		normalizedModuleVersion = &stripped
	}

	ws, err := dbClient.Workspaces.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace (ID %s) associated with run", workspaceID)
	}
	if ws == nil {
		return nil, errors.New("failed to get workspace associated with run", errors.WithErrorCode(errors.ENotFound))
	}

	moduleRegistrySource, err := moduleResolver.ParseModuleRegistrySource(
		ctx, *moduleSource, tokenGetter, GetFederatedRegistry(dbClient, ws))
	if err != nil && err != registry.ErrRemoteModuleSource {
		return nil, errors.Wrap(err, "failed to resolve module source", errors.WithErrorCode(errors.EInvalid))
	}

	// registry source is nil for a remote module source that doesn't use the registry protocol.
	if moduleRegistrySource == nil {
		return resolved, nil
	}
	resolved.Source = moduleRegistrySource

	module, err := moduleRegistrySource.LocalRegistryModule(ctx)
	if err != nil {
		return nil, err
	}
	// If the module is private, verify the caller is authorized to use it.
	if module != nil && module.Private {
		caller, cErr := auth.AuthorizeCaller(ctx)
		if cErr != nil {
			return nil, cErr
		}
		if err := caller.RequireAccessToInheritableResource(
			ctx, modeltypes.TerraformModuleModelType, auth.WithGroupID(module.GroupID)); err != nil {
			return nil, errors.Wrap(err, "caller not authorized to use module %s", *moduleSource)
		}
	}

	resolvedVersion, err := moduleRegistrySource.ResolveSemanticVersion(ctx, normalizedModuleVersion, includeModulePrereleases)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve module source", errors.WithErrorCode(errors.EInvalid))
	}
	resolved.Version = &resolvedVersion

	resolved.Digest, err = moduleRegistrySource.ResolveDigest(ctx, resolvedVersion)
	if err != nil {
		return nil, err
	}

	return resolved, nil
}
