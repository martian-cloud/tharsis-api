package models

import (
	"time"
)

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

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (w *WorkspaceAssessment) ResolveMetadata(key string) (string, error) {
	val, err := w.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "started_at":
			val = w.StartedAtTimestamp.Format(time.RFC3339Nano)
		default:
			return "", err
		}
	}

	return val, nil
}
