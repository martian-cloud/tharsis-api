package models

import "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"

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

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (a *Apply) ResolveMetadata(key string) (string, error) {
	return a.Metadata.resolveFieldValue(key)
}

// Validate returns an error if the model is not valid
func (a *Apply) Validate() error {
	if a.ErrorMessage != nil && a.Status != ApplyErrored {
		return errors.New("invalid apply status, must be errored if error message is set", errors.WithErrorCode(errors.EInvalid))
	}
	return nil
}
