package models

import (
	"time"

	"github.com/aws/smithy-go/ptr"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*WorkspaceAssessment)(nil)

// WorkspaceAssessment represents a workspace assessment to check for drift
type WorkspaceAssessment struct {
	Metadata             ResourceMetadata
	StartedAtTimestamp   time.Time
	CompletedAtTimestamp *time.Time
	HasDrift             bool
	RunID                *string
	WorkspaceID          string
	RequiresNotification bool
}

// GetID returns the Metadata ID.
func (w *WorkspaceAssessment) GetID() string {
	return w.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (w *WorkspaceAssessment) GetGlobalID() string {
	return gid.ToGlobalID(w.GetModelType(), w.Metadata.ID)
}

// GetModelType returns the type of the model.
func (w *WorkspaceAssessment) GetModelType() types.ModelType {
	return types.WorkspaceAssessmentModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (w *WorkspaceAssessment) ResolveMetadata(key string) (*string, error) {
	val, err := w.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "started_at":
			return ptr.String(w.StartedAtTimestamp.Format(time.RFC3339Nano)), nil
		default:
			return nil, err
		}
	}

	return val, nil
}

// Validate validates the model.
func (w *WorkspaceAssessment) Validate() error {
	return nil
}
