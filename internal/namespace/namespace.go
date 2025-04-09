package namespace

import "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"

// Namespace represents a group or workspace
type Namespace interface {
	GetID() string
	GetPath() string
	GetParentID() string
	ExpandPath() []string
	GetRunnerTags() []string
	DriftDetectionEnabled() *bool
	ResolveMetadata(key string) (string, error)
	GetResourceType() permissions.ResourceType
}
