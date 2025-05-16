package resolver

import (
	graphql "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

/* Resolver for state version outputs, for queries initiated on workspaces. */

// StateVersionOutputResolver resolves a state version output resource
type StateVersionOutputResolver struct {
	stateVersionOutput *models.StateVersionOutput
}

// ID resolver
func (r *StateVersionOutputResolver) ID() graphql.ID {
	return graphql.ID(r.stateVersionOutput.GetGlobalID())
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
