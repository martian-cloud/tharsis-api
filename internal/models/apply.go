package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

var _ Model = (*Apply)(nil)

// ApplyStatus represents the various states for a Apply resource
type ApplyStatus string

// Apply Status Types
const (
	ApplyCanceled ApplyStatus = "canceled"
	ApplyCreated  ApplyStatus = "created"
	ApplyErrored  ApplyStatus = "errored"
	ApplyFinished ApplyStatus = "finished"
	ApplyPending  ApplyStatus = "pending"
	ApplyQueued   ApplyStatus = "queued"
	ApplyRunning  ApplyStatus = "running"
)

// Apply includes information related to running a terraform plan command
type Apply struct {
	WorkspaceID  string
	Status       ApplyStatus
	TriggeredBy  string
	Comment      string
	Metadata     ResourceMetadata
	ErrorMessage *string
}

// GetID returns the ID of the Apply resource
func (a *Apply) GetID() string {
	return a.Metadata.ID
}

// GetGlobalID returns the GID of the Apply resource
func (a *Apply) GetGlobalID() string {
	return gid.ToGlobalID(a.GetModelType(), a.Metadata.ID)
}

// GetModelType returns the Model's type
func (a *Apply) GetModelType() types.ModelType {
	return types.ApplyModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (a *Apply) ResolveMetadata(key string) (*string, error) {
	return a.Metadata.resolveFieldValue(key)
}

// Validate returns an error if the model is not valid
func (a *Apply) Validate() error {
	if a.ErrorMessage != nil && a.Status != ApplyErrored {
		return errors.New("invalid apply status, must be errored if error message is set", errors.WithErrorCode(errors.EInvalid))
	}
	return nil
}
