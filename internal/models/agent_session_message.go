package models

import (
	"encoding/json"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*AgentSessionMessage)(nil)

// AgentSessionMessage represents a single message in an agent session's conversation history.
type AgentSessionMessage struct {
	Metadata  ResourceMetadata
	SessionID string
	RunID     string
	ParentID  *string
	Role      string
	Content   json.RawMessage
}

// GetID returns the Metadata ID.
func (a *AgentSessionMessage) GetID() string {
	return a.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (a *AgentSessionMessage) GetGlobalID() string {
	return gid.ToGlobalID(a.GetModelType(), a.Metadata.ID)
}

// GetModelType returns the model type.
func (a *AgentSessionMessage) GetModelType() types.ModelType {
	return types.AgentSessionMessageModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (a *AgentSessionMessage) ResolveMetadata(key string) (*string, error) {
	return a.Metadata.resolveFieldValue(key)
}

// Validate validates the model.
func (a *AgentSessionMessage) Validate() error {
	return nil
}
