package models

import (
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
)

// RunnerType constant
type RunnerType string

// RunnerType constants
const (
	GroupRunnerType  RunnerType = "group"
	SharedRunnerType RunnerType = "shared"
)

// Runner resource
type Runner struct {
	Type         RunnerType
	Name         string
	Description  string
	GroupID      *string
	ResourcePath string
	CreatedBy    string
	Metadata     ResourceMetadata
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
		return errors.NewError(errors.EInvalid, "runner type must be specified")
	}

	if r.Type == SharedRunnerType && r.GroupID != nil {
		return errors.NewError(errors.EInvalid, "shared runner should not have a group specified")
	}

	if r.Type == GroupRunnerType && r.GroupID == nil {
		return errors.NewError(errors.EInvalid, "group runner must specify a group")
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
