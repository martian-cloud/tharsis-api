package models

import (
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// RunnerType constant
type RunnerType string

// RunnerType constants
const (
	GroupRunnerType  RunnerType = "group"
	SharedRunnerType RunnerType = "shared"
)

// Equals returns true if the runner type is equal to the other runner type
func (r RunnerType) Equals(other RunnerType) bool {
	return r == other
}

// Runner resource
type Runner struct {
	Type         RunnerType
	Name         string
	Description  string
	GroupID      *string
	ResourcePath string
	CreatedBy    string
	Metadata     ResourceMetadata
	Disabled     bool
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (r *Runner) ResolveMetadata(key string) (string, error) {
	return r.Metadata.resolveFieldValue(key)
}

// Validate returns an error if the model is not valid
func (r *Runner) Validate() error {
	// Verify name satisfies constraints
	if err := verifyValidName(r.Name); err != nil {
		return err
	}

	if err := verifyValidDescription(r.Description); err != nil {
		return err
	}

	if r.Type == "" {
		return errors.New("runner type must be specified", errors.WithErrorCode(errors.EInvalid))
	}

	if r.Type == SharedRunnerType && r.GroupID != nil {
		return errors.New("shared runner should not have a group specified", errors.WithErrorCode(errors.EInvalid))
	}

	if r.Type == GroupRunnerType && r.GroupID == nil {
		return errors.New("group runner must specify a group", errors.WithErrorCode(errors.EInvalid))
	}

	return nil
}

// GetGroupPath returns the group path
func (r *Runner) GetGroupPath() string {
	if r.Type == SharedRunnerType {
		return ""
	}
	return r.ResourcePath[:strings.LastIndex(r.ResourcePath, "/")]
}
