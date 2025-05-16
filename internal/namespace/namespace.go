package namespace

import "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"

// Namespace represents a group or workspace
type Namespace interface {
	models.Model // This interface must be implemented by any type that implements Namespace
	GetPath() string
	GetParentID() string
	ExpandPath() []string
	GetRunnerTags() []string
	DriftDetectionEnabled() *bool
}
