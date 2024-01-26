package models

// LogStream represents a stream of logs
type LogStream struct {
	JobID           *string
	RunnerSessionID *string
	Metadata        ResourceMetadata
	Size            int
	Completed       bool
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (o *LogStream) ResolveMetadata(key string) (string, error) {
	return o.Metadata.resolveFieldValue(key)
}
