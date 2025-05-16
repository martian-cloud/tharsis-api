package resolver

import (
	"context"
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// Key type is used for attaching resolver state to the context
type key string

const (
	resolverStateKey key = "resolverState"
)

// State contains the services required by resolvers
type State struct {
	Config         *config.Config
	Logger         logger.Logger
	ServiceCatalog *services.Catalog
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

func getServiceCatalog(ctx context.Context) *services.Catalog {
	return extract(ctx).ServiceCatalog
}

// nolint
func getLogger(ctx context.Context) logger.Logger {
	return extract(ctx).Logger
}

func getConfig(ctx context.Context) *config.Config {
	return extract(ctx).Config
}

// toModelID resolves the ID for a model type from its path or GID
func toModelID(ctx context.Context, path *string, globalID *string, modelType types.ModelType) (string, error) {
	var valueToResolve string
	switch {
	case path != nil && globalID != nil:
		return "", errors.New(fmt.Sprintf("cannot specify both id and path for %s", modelType.Name()), errors.WithErrorCode(errors.EInvalid))
	case path != nil:
		valueToResolve = modelType.BuildTRN(*path)
	case globalID != nil:
		valueToResolve = *globalID
	default:
		return "", errors.New(fmt.Sprintf("either id or path must be specified for %s", modelType.Name()), errors.WithErrorCode(errors.EInvalid))
	}

	return getServiceCatalog(ctx).FetchModelID(ctx, valueToResolve)
}
