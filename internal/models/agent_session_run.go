package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*AgentSessionRun)(nil)

// AgentSessionRunStatus represents the status of an agent session run.
type AgentSessionRunStatus string

// AgentSessionRunStatus constants
const (
	AgentSessionRunRunning   AgentSessionRunStatus = "running"
	AgentSessionRunFinished  AgentSessionRunStatus = "finished"
	AgentSessionRunErrored   AgentSessionRunStatus = "errored"
	AgentSessionRunCancelled AgentSessionRunStatus = "canceled"
)

// AgentSessionRun represents a single run within an agent session.
type AgentSessionRun struct {
	Metadata        ResourceMetadata
	SessionID       string
	PreviousRunID   *string
	LastMessageID   *string
	Status          AgentSessionRunStatus
	ErrorMessage    *string
	CancelRequested bool
}

// GetID returns the Metadata ID.
func (a *AgentSessionRun) GetID() string {
	return a.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (a *AgentSessionRun) GetGlobalID() string {
	return gid.ToGlobalID(a.GetModelType(), a.Metadata.ID)
}

// GetModelType returns the model type.
func (a *AgentSessionRun) GetModelType() types.ModelType {
	return types.AgentSessionRunModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (a *AgentSessionRun) ResolveMetadata(key string) (*string, error) {
	return a.Metadata.resolveFieldValue(key)
}

// Validate validates the model.
func (a *AgentSessionRun) Validate() error {
	return nil
}
