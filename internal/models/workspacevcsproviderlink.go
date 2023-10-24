package models

import (
	"path/filepath"
	"regexp"

	"github.com/bmatcuk/doublestar/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// maxPatternLength defines the maximum length a regex or glob pattern can be.
const maxPatternLength = 30

var (
	// errInvalidPatternLength indicates when a pattern is either empty
	// or exceeds maxPatternLength.
	errInvalidPatternLength = errors.New(
		"Invalid glob pattern or regex, must be non-empty and no larger than %d characters",
		maxPatternLength,
		errors.WithErrorCode(errors.EInvalid),
	)

	// errInvalidPattern is a generic error indicating either an invalid
	// glob pattern or regex.
	errInvalidPattern = errors.New(
		"Invalid glob pattern or regex",
		errors.WithErrorCode(errors.EInvalid),
	)
)

// WorkspaceVCSProviderLink represents a link for a
// version control system provider to a workspace.
type WorkspaceVCSProviderLink struct {
	CreatedBy           string
	WorkspaceID         string
	ProviderID          string
	TokenNonce          string
	RepositoryPath      string
	WebhookID           string   // Webhook ID if Tharsis configured it.
	ModuleDirectory     *string  // Path to Terraform module, otherwise repo root.
	Branch              string   // A branch name to filter on.
	TagRegex            *string  // A tag regex to use as a filter.
	GlobPatterns        []string // Glob patterns to use for monitoring changes.
	Metadata            ResourceMetadata
	AutoSpeculativePlan bool // Whether to create speculative plans automatically for PRs.
	WebhookDisabled     bool
}

// Validate verifies a VCS Provider link struct.
func (wpl *WorkspaceVCSProviderLink) Validate() error {
	// Verify glob patterns.
	for _, pattern := range wpl.GlobPatterns {
		if len(pattern) > maxPatternLength {
			return errInvalidPatternLength
		}

		if !doublestar.ValidatePattern(filepath.ToSlash(pattern)) {
			return errInvalidPattern
		}
	}

	// Verify tag regex.
	if wpl.TagRegex != nil {
		if len(*wpl.TagRegex) > maxPatternLength {
			return errInvalidPatternLength
		}

		if _, err := regexp.Compile(*wpl.TagRegex); err != nil {
			return errInvalidPattern
		}
	}

	return nil
}
