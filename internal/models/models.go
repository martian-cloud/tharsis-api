package models

import (
	"fmt"
	"regexp"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
)

// maxDescriptionLength is the maximum length for a resource's description field.
const maxDescriptionLength int = 100

// nameRegex allows letters, numbers with - and _ allowed in non leading or trailing positions, max length is 64
var nameRegex = regexp.MustCompile("^[0-9a-z](?:[0-9a-z-_]{0,62}[0-9a-z])?$")

// ResourceMetadata contains metadata for a particular resource
type ResourceMetadata struct {
	CreationTimestamp    *time.Time `json:"createdAt"`
	LastUpdatedTimestamp *time.Time `json:"updatedAt,omitempty" `
	ID                   string     `json:"id"`
	Version              int        `json:"version"`
}

func verifyValidName(name string) error {
	if !nameRegex.MatchString(name) {
		return errors.NewError(errors.EInvalid, "Invalid name, name can only include lowercase letters and numbers with - and _ supported "+
			"in non leading or trailing positions. Max length is 64 characters.")
	}
	return nil
}

func verifyValidDescription(description string) error {
	if len(description) > maxDescriptionLength {
		return errors.NewError(errors.EInvalid, fmt.Sprintf("Invalid description, cannot be greater than %d characters", maxDescriptionLength))
	}
	return nil
}
