package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// PlanStatus represents the various states for a Plan resource
type PlanStatus string

// Run Status Types
const (
	PlanCanceled PlanStatus = "canceled"
	PlanQueued   PlanStatus = "queued"
	PlanErrored  PlanStatus = "errored"
	PlanFinished PlanStatus = "finished"
	PlanPending  PlanStatus = "pending"
	PlanRunning  PlanStatus = "running"
)

// PlanSummary contains a summary of the types of changes this plan includes
type PlanSummary struct {
	ResourceAdditions    int32
	ResourceChanges      int32
	ResourceDestructions int32
	ResourceImports      int32
	ResourceDrift        int32
	OutputAdditions      int32
	OutputChanges        int32
	OutputDestructions   int32
}

// Plan includes information related to running a terraform plan command
type Plan struct {
	ErrorMessage *string
	WorkspaceID  string
	Status       PlanStatus
	Metadata     ResourceMetadata
	PlanDiffSize int
	Summary      PlanSummary
	HasChanges   bool
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (p *Plan) ResolveMetadata(key string) (string, error) {
	return p.Metadata.resolveFieldValue(key)
}

// Validate returns an error if the model is not valid
func (p *Plan) Validate() error {
	if p.ErrorMessage != nil && p.Status != PlanErrored {
		return errors.New("invalid plan status, must be errored if error message is set", errors.WithErrorCode(errors.EInvalid))
	}
	return nil
}
