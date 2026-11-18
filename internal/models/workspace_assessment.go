package models

import (
	"time"
	"unicode/utf8"

	"github.com/aws/smithy-go/ptr"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/sanitize"
)

var _ Model = (*WorkspaceAssessment)(nil)

// AssessmentStaleTimeout is how long an in-progress assessment (CompletedAtTimestamp
// nil) may go without an update before it is treated as abandoned and eligible to be
// restarted. Completion is event-driven (set when the run reaches a terminal state);
// a run orphaned before reaching one would otherwise leave the assessment in progress
// forever, permanently blocking the workspace from being reassessed.
const AssessmentStaleTimeout = time.Hour

// MaxAssessmentErrorMessageLength caps the length in bytes of a stored assessment error
// message. A module resolution failure can embed a third-party registry's response body in
// its message, so the text is bounded before it reaches the database.
const MaxAssessmentErrorMessageLength = 1024

// assessmentErrorMessageEllipsis marks a message shortened by TruncateAssessmentErrorMessage.
const assessmentErrorMessageEllipsis = "..."

// WorkspaceAssessment represents a workspace assessment to check for drift
type WorkspaceAssessment struct {
	Metadata             ResourceMetadata
	StartedAtTimestamp   time.Time
	CompletedAtTimestamp *time.Time
	HasDrift             bool
	RunID                *string
	WorkspaceID          string
	RequiresNotification bool
	// ErrorMessage records why the last attempt could not create an assessment run: the
	// module was renamed or deleted, or its registry credential expired. Non-nil means it
	// failed; cleared when a run is created again.
	//
	// Accepted trade-off: this text can originate outside Tharsis, since both failure cases
	// wrap the external registry's response. Readable via GraphQL by anyone with
	// workspace-view access, the same breadth accepted for Plan and Apply errorMessage.
	// Whoever sets the module source controls what every viewer sees.
	ErrorMessage *string
	// AvailableAtTimestamp is when the scheduler should reconsider this workspace; relies on
	// this column instead of recomputing from CompletedAtTimestamp. Stamped on every
	// completion, regardless of whether the run was scheduler- or user-initiated: base
	// interval for success or a verdict-less run, further out per consecutive failure.
	// Cleared to nil on a new run to avoid re-selecting the workspace mid-run.
	AvailableAtTimestamp *time.Time
	// ConsecutiveFailures counts attempts in a row that failed to create a run (the conditions
	// in ErrorMessage). It drives the AvailableAtTimestamp backoff and resets to zero once a run
	// is created again. A run created but failed during planning is a different, already-retried
	// failure, and doesn't count.
	ConsecutiveFailures int
}

// IsStaleInProgress reports whether the assessment is in progress but has not been
// updated within AssessmentStaleTimeout, indicating its run was abandoned.
func (w *WorkspaceAssessment) IsStaleInProgress() bool {
	return w.CompletedAtTimestamp == nil &&
		w.Metadata.LastUpdatedTimestamp != nil &&
		time.Since(*w.Metadata.LastUpdatedTimestamp) >= AssessmentStaleTimeout
}

// GetID returns the Metadata ID.
func (w *WorkspaceAssessment) GetID() string {
	return w.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (w *WorkspaceAssessment) GetGlobalID() string {
	return gid.ToGlobalID(w.GetModelType(), w.Metadata.ID)
}

// GetModelType returns the type of the model.
func (w *WorkspaceAssessment) GetModelType() types.ModelType {
	return types.WorkspaceAssessmentModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (w *WorkspaceAssessment) ResolveMetadata(key string) (*string, error) {
	val, err := w.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "started_at":
			return ptr.String(w.StartedAtTimestamp.Format(time.RFC3339Nano)), nil
		default:
			return nil, err
		}
	}

	return val, nil
}

// Validate validates the model.
func (w *WorkspaceAssessment) Validate() error {
	if w.ErrorMessage != nil && len(*w.ErrorMessage) > MaxAssessmentErrorMessageLength {
		return errors.New(
			"assessment error message cannot be greater than %d characters",
			MaxAssessmentErrorMessageLength,
			errors.WithErrorCode(errors.EInvalid),
		)
	}

	return nil
}

// TruncateAssessmentErrorMessage bounds msg to MaxAssessmentErrorMessageLength bytes and
// coerces it to valid UTF-8, reusing pkg/sanitize (see corerun.TruncateAndSanitizeErrorMessage).
// Sanitizing first matters: msg can have multiple runs of invalid bytes, each growing from
// 1 byte to the 3-byte replacement character. Cutting raw bytes first, with a fixed margin
// for growth, only accounts for one such run at the cut point. Cutting the now-valid result
// at a rune boundary can't grow it further, so no margin is needed. A shortened message
// gets an ellipsis.
func TruncateAssessmentErrorMessage(msg string) string {
	sanitized := sanitize.ToValidUTF8(msg)
	if len(sanitized) <= MaxAssessmentErrorMessageLength {
		return sanitized
	}

	contentLimit := MaxAssessmentErrorMessageLength - len(assessmentErrorMessageEllipsis)
	cutAt := contentLimit
	for cutAt > 0 && !utf8.RuneStart(sanitized[cutAt]) {
		cutAt--
	}
	return sanitized[:cutAt] + assessmentErrorMessageEllipsis
}
