package models

import "time"

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

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (a *RunnerSession) ResolveMetadata(key string) (string, error) {
	val, err := a.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "last_contacted_at":
			val = a.LastContactTimestamp.Format(time.RFC3339Nano)
		default:
			return "", err
		}
	}

	return val, nil
}

// Active returns true if the session has received a heartbeat within the last heartbeat interval
func (a *RunnerSession) Active() bool {
	// Check if the elapsed time since the last heartbeat exceeds the heartbeat interval plus some leeway
	return time.Since(a.LastContactTimestamp) <= (RunnerSessionHeartbeatInterval + (5 * time.Second))
}
