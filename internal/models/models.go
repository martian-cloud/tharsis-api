package models

import (
	"fmt"
	"regexp"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

const (
	// maxDescriptionLength is the maximum length for a resource's description field.
	maxDescriptionLength = 100
	maxTagsPerResource   = 10
)

// nameRegex allows letters, numbers with - and _ allowed in non leading or trailing positions, max length is 64
var nameRegex = regexp.MustCompile("^[0-9a-z](?:[0-9a-z-_]{0,62}[0-9a-z])?$")

// ResourceMetadata contains metadata for a particular resource
type ResourceMetadata struct {
	CreationTimestamp    *time.Time `json:"createdAt"`
	LastUpdatedTimestamp *time.Time `json:"updatedAt,omitempty" `
	ID                   string     `json:"id"`
	Version              int        `json:"version"`
}

// resolveFieldValue resolves metadata field values for cursor based pagination
func (r *ResourceMetadata) resolveFieldValue(key string) (string, error) {
	var resp string

	switch key {
	case "id":
		resp = r.ID
	case "updated_at":
		resp = r.LastUpdatedTimestamp.Format(time.RFC3339Nano)
	case "created_at":
		resp = r.CreationTimestamp.Format(time.RFC3339Nano)
	default:
		return "", fmt.Errorf("invalid field key requested: %s", key)
	}

	return resp, nil
}

func verifyValidName(name string) error {
	if !nameRegex.MatchString(name) {
		return errors.New("Invalid name, name can only include lowercase letters and numbers with - and _ supported "+
			"in non leading or trailing positions. Max length is 64 characters.", errors.WithErrorCode(errors.EInvalid))
	}
	return nil
}

func verifyValidDescription(description string) error {
	if len(description) > maxDescriptionLength {
		return errors.New("invalid description, cannot be greater than %d characters", maxDescriptionLength, errors.WithErrorCode(errors.EInvalid))
	}
	return nil
}
