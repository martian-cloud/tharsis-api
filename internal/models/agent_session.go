package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*AgentSession)(nil)

// AgentSession represents a persistent agent conversation session.
type AgentSession struct {
	Metadata     ResourceMetadata
	UserID       string
	TotalCredits float64
}

// GetID returns the Metadata ID.
func (a *AgentSession) GetID() string {
	return a.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (a *AgentSession) GetGlobalID() string {
	return gid.ToGlobalID(a.GetModelType(), a.Metadata.ID)
}

// GetModelType returns the model type.
func (a *AgentSession) GetModelType() types.ModelType {
	return types.AgentSessionModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (a *AgentSession) ResolveMetadata(key string) (*string, error) {
	return a.Metadata.resolveFieldValue(key)
}

// Validate validates the model.
func (a *AgentSession) Validate() error {
	return nil
}
