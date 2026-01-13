package namespace

import "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"

// Type represents the type of namespace
type Type string

const (
	// TypeGroup represents a group namespace
	TypeGroup Type = "GROUP"
	// TypeWorkspace represents a workspace namespace
	TypeWorkspace Type = "WORKSPACE"
)

// Namespace represents a group or workspace
type Namespace interface {
	models.Model // This interface must be implemented by any type that implements Namespace
	GetPath() string
	GetParentID() string
	ExpandPath() []string
	GetRunnerTags() []string
	DriftDetectionEnabled() *bool
}
