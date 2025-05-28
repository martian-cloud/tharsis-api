package models

import (
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

var _ Model = (*Runner)(nil)

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
	Type            RunnerType
	Name            string
	Description     string
	GroupID         *string
	CreatedBy       string
	Metadata        ResourceMetadata
	Disabled        bool
	Tags            []string
	RunUntaggedJobs bool
}

// GetID returns the Metadata ID.
func (r *Runner) GetID() string {
	return r.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (r *Runner) GetGlobalID() string {
	return gid.ToGlobalID(r.GetModelType(), r.Metadata.ID)
}

// GetModelType returns the model type.
func (r *Runner) GetModelType() types.ModelType {
	return types.RunnerModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (r *Runner) ResolveMetadata(key string) (*string, error) {
	val, err := r.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "group_path":
			path := r.GetGroupPath()
			return &path, nil
		default:
			return nil, err
		}
	}

	return val, nil
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

	if !r.RunUntaggedJobs && len(r.Tags) == 0 {
		return errors.New("at least one tag must be specified when the run untagged job setting is set to false",
			errors.WithErrorCode(errors.EInvalid))
	}

	// Check for duplicate tags, too-long tags, and too many tags.
	return verifyValidRunnerTags(r.Tags)
}

// GetResourcePath returns the resource path
func (r *Runner) GetResourcePath() string {
	return strings.Split(r.Metadata.TRN[len(types.TRNPrefix):], ":")[1]
}

// GetGroupPath returns the group path
func (r *Runner) GetGroupPath() string {
	if r.Type == SharedRunnerType {
		return ""
	}

	resourcePath := r.GetResourcePath()
	return resourcePath[:strings.LastIndex(resourcePath, "/")]
}

// verifyValidRunnerTags checks for duplicate tags, too-long tags, and too many tags.
func verifyValidRunnerTags(tags []string) error {
	if tags == nil {
		return nil
	}

	if len(tags) > maxTagsPerResource {
		return errors.New("exceeded max number of tags per resource: %d", maxTagsPerResource,
			errors.WithErrorCode(errors.EInvalid))
	}

	tagMap := map[string]struct{}{}
	for _, tag := range tags {
		if _, ok := tagMap[tag]; ok {
			return errors.New("duplicate tag values are not allowed: tag %s has been duplicated",
				tag, errors.WithErrorCode(errors.EInvalid))
		}
		tagMap[tag] = struct{}{}

		if !nameRegex.MatchString(tag) {
			return errors.New("tag %s contains invalid characters, only lowercase letters and numbers with - and _ supported "+
				"in non leading or trailing positions. Max length is 64 characters.", tag, errors.WithErrorCode(errors.EInvalid))
		}
	}

	return nil
}
