package models

// ResourceLimit represents a resource limit
type ResourceLimit struct {
	Name     string
	Metadata ResourceMetadata
	Value    int
}
