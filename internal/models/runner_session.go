package models

import (
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*RunnerSession)(nil)

// RunnerSessionHeartbeatInterval is the interval that runners should send heartbeats
const RunnerSessionHeartbeatInterval = time.Minute

// RunnerSession represents a session for a runner.
type RunnerSession struct {
	LastContactTimestamp time.Time
	RunnerID             string
	Metadata             ResourceMetadata
	ErrorCount           int
	Internal             bool
}

// GetID returns the Metadata ID.
func (r *RunnerSession) GetID() string {
	return r.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (r *RunnerSession) GetGlobalID() string {
	return gid.ToGlobalID(r.GetModelType(), r.Metadata.ID)
}

// GetModelType returns the model type.
func (r *RunnerSession) GetModelType() types.ModelType {
	return types.RunnerSessionModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (r *RunnerSession) ResolveMetadata(key string) (string, error) {
	val, err := r.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "last_contacted_at":
			val = r.LastContactTimestamp.Format(time.RFC3339Nano)
		default:
			return "", err
		}
	}

	return val, nil
}

// Validate validates the model.
func (r *RunnerSession) Validate() error {
	return nil
}

// Active returns true if the session has received a heartbeat within the last heartbeat interval
func (r *RunnerSession) Active() bool {
	// Check if the elapsed time since the last heartbeat exceeds the heartbeat interval plus some leeway
	return time.Since(r.LastContactTimestamp) <= (RunnerSessionHeartbeatInterval + (5 * time.Second))
}
