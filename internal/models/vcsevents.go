package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*VCSEvent)(nil)

// VCSEventStatus defines an enum that represents the status of a VCS event.
type VCSEventStatus string

// VCSEventType defines an enum that represents the type of VCS event.
type VCSEventType string

// Equals is a convenience func that returns whether two events are equal.
func (have VCSEventType) Equals(want VCSEventType) bool {
	return have == want
}

// VCSEventStatus constants.
const (
	VCSEventPending  VCSEventStatus = "pending"
	VCSEventFinished VCSEventStatus = "finished"
	VCSEventErrored  VCSEventStatus = "errored"
)

// VCSEventType constants.
const (
	BranchEventType       VCSEventType = "branch"
	TagEventType          VCSEventType = "tag"
	MergeRequestEventType VCSEventType = "merge_request"
	ManualEventType       VCSEventType = "manual"
)

// VCSEvent represents a vcs event that result in
// configuration changes via Tharsis.
type VCSEvent struct {
	ErrorMessage        *string // An error message indicating the reason event failed.
	CommitID            *string // Commit ID associated with this event.
	SourceReferenceName *string // Name of branch or tag that triggered this event.
	WorkspaceID         string
	RepositoryURL       string
	Type                VCSEventType
	Status              VCSEventStatus
	Metadata            ResourceMetadata
}

// GetID returns the Metadata ID.
func (v *VCSEvent) GetID() string {
	return v.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (v *VCSEvent) GetGlobalID() string {
	return gid.ToGlobalID(v.GetModelType(), v.Metadata.ID)
}

// GetModelType returns the type of the model.
func (v *VCSEvent) GetModelType() types.ModelType {
	return types.VCSEventModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (v *VCSEvent) ResolveMetadata(key string) (*string, error) {
	return v.Metadata.resolveFieldValue(key)
}

// Validate validates the model.
func (v *VCSEvent) Validate() error {
	return nil
}
