package models

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestWorkspaceAssessment_IsStaleInProgress(t *testing.T) {
	now := time.Now()
	old := now.Add(-2 * AssessmentStaleTimeout)
	recent := now.Add(-AssessmentStaleTimeout / 2)
	completed := now.Add(-time.Minute)

	testCases := []struct {
		name        string
		lastUpdated *time.Time
		completedAt *time.Time
		expect      bool
	}{
		{
			name:        "in progress and not updated within the stale timeout",
			lastUpdated: &old,
			completedAt: nil,
			expect:      true,
		},
		{
			name:        "in progress but updated recently",
			lastUpdated: &recent,
			completedAt: nil,
			expect:      false,
		},
		{
			name:        "completed assessment is never stale-in-progress",
			lastUpdated: &old,
			completedAt: &completed,
			expect:      false,
		},
		{
			name:        "missing last-updated timestamp is not treated as stale",
			lastUpdated: nil,
			completedAt: nil,
			expect:      false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			wa := &WorkspaceAssessment{
				Metadata:             ResourceMetadata{LastUpdatedTimestamp: test.lastUpdated},
				CompletedAtTimestamp: test.completedAt,
			}
			assert.Equal(t, test.expect, wa.IsStaleInProgress())
		})
	}
}

func TestWorkspaceAssessment_Validate(t *testing.T) {
	testCases := []struct {
		name            string
		errorMessage    *string
		expectErrorCode errors.CodeType
	}{
		{
			name:         "no error message is valid",
			errorMessage: nil,
		},
		{
			name:         "error message within the cap is valid",
			errorMessage: ptr.String("module not found for source registry/example/aws"),
		},
		{
			name:         "error message exactly at the cap is valid",
			errorMessage: ptr.String(strings.Repeat("a", MaxAssessmentErrorMessageLength)),
		},
		{
			name:            "error message over the cap is rejected",
			errorMessage:    ptr.String(strings.Repeat("a", MaxAssessmentErrorMessageLength+1)),
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			wa := &WorkspaceAssessment{ErrorMessage: test.errorMessage}

			err := wa.Validate()

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			assert.Nil(t, err)
		})
	}
}

func TestTruncateAssessmentErrorMessage(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectShorten  bool
		expectUnchange bool
	}{
		{
			name:           "short message is returned unchanged",
			input:          "module not found for source registry/example/aws",
			expectUnchange: true,
		},
		{
			name:           "message exactly at the cap is returned unchanged",
			input:          strings.Repeat("a", MaxAssessmentErrorMessageLength),
			expectUnchange: true,
		},
		{
			// Sanitizing first means msg is already valid UTF-8 by the time it's cut, so this
			// proves the cut backs up to a rune boundary rather than splitting the 2-byte rune
			// placed so its first byte would otherwise land exactly at the naive cut point.
			name:          "a rune straddling the cut point is excluded whole, not split",
			input:         strings.Repeat("x", MaxAssessmentErrorMessageLength-len(assessmentErrorMessageEllipsis)) + "é" + strings.Repeat("y", 50),
			expectShorten: true,
		},
		{
			// Sanitizing growth is not limited to the cut point: msg can carry more than one
			// run of invalid bytes anywhere within it, and each is replaced independently.
			// Several lone, otherwise-invalid bytes well before the cap, each becoming a
			// 3-byte replacement character, would have exceeded the cap under a fixed margin
			// sized for only one such run.
			name:          "multiple runs of invalid bytes before the cap still fit after truncation",
			input:         strings.Repeat("\x80", 10) + strings.Repeat("a", MaxAssessmentErrorMessageLength),
			expectShorten: true,
		},
		{
			name:          "message over the cap is shortened",
			input:         strings.Repeat("a", MaxAssessmentErrorMessageLength+1),
			expectShorten: true,
		},
		{
			name:          "much longer message is shortened",
			input:         strings.Repeat("a", MaxAssessmentErrorMessageLength*10),
			expectShorten: true,
		},
		{
			// A registry's 401 body is interpolated into the message verbatim, so it may contain
			// multibyte characters that must not be split at the truncation boundary.
			name:          "multibyte message is shortened without splitting a character",
			input:         strings.Repeat("é", MaxAssessmentErrorMessageLength),
			expectShorten: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			got := TruncateAssessmentErrorMessage(test.input)

			if test.expectUnchange {
				assert.Equal(t, test.input, got)
			}

			if test.expectShorten {
				assert.NotEqual(t, test.input, got)
				assert.True(t, strings.HasSuffix(got, "..."), "a shortened message is marked with an ellipsis")
			}

			assert.LessOrEqual(t, len(got), MaxAssessmentErrorMessageLength,
				"the result must fit the column and satisfy Validate")
			assert.True(t, utf8.ValidString(got), "the result must remain valid UTF-8")

			// The whole point of truncating is that the result is persistable.
			wa := &WorkspaceAssessment{ErrorMessage: &got}
			require.Nil(t, wa.Validate())
		})
	}
}
