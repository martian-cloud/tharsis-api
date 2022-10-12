package resolver

import (
	"context"

	graphql "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

/* Resolver for state version outputs, for queries initiated on workspaces. */

// StateVersionOutputResolver resolves a state version output resource
type StateVersionOutputResolver struct {
	stateVersionOutput *models.StateVersionOutput
}

// ID resolver
func (r *StateVersionOutputResolver) ID() graphql.ID {
	return graphql.ID(gid.ToGlobalID(gid.StateVersionOutputType, r.stateVersionOutput.Metadata.ID))
}

// Name resolver
func (r *StateVersionOutputResolver) Name() string {
	return r.stateVersionOutput.Name
}

// Value resolver
func (r *StateVersionOutputResolver) Value() string {
	return string(r.stateVersionOutput.Value)
}

// Type resolver
func (r *StateVersionOutputResolver) Type() string {
	return string(r.stateVersionOutput.Type)
}

// Sensitive resolver
func (r *StateVersionOutputResolver) Sensitive() bool {
	return r.stateVersionOutput.Sensitive
}

// Metadata resolver
func (r *StateVersionOutputResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.stateVersionOutput.Metadata}
}

/* State Version Output Queries */

func getStateVersionOutputs(ctx context.Context,
	stateVersionID string) ([]*StateVersionOutputResolver, error) {

	result, err := getWorkspaceService(ctx).GetStateVersionOutputs(ctx, stateVersionID)
	if err != nil {
		return nil, err
	}

	// Make a new list of resolvers with _copies_ of the state version outputs.
	// ... following the example of variables.
	resolvers := []*StateVersionOutputResolver{}
	for _, v := range result {
		// Must make a copy of the output, lest all returned resolvers be the same.
		outputCopy := v
		resolvers = append(resolvers, &StateVersionOutputResolver{stateVersionOutput: &outputCopy})
	}

	return resolvers, nil
}

// The End.
