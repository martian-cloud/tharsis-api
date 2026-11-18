package resolver

import (
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// TestWorkspaceAssessmentResolver_ErrorMessage covers the field that surfaces why drift
// detection stopped producing verdicts for a workspace. Null means the last attempt did not
// fail, so the UI must be able to distinguish absent from empty.
func TestWorkspaceAssessmentResolver_ErrorMessage(t *testing.T) {
	t.Run("no recorded failure resolves to null", func(t *testing.T) {
		r := &WorkspaceAssessmentResolver{assessment: &models.WorkspaceAssessment{}}

		assert.Nil(t, r.ErrorMessage())
	})

	t.Run("a recorded failure resolves to its message", func(t *testing.T) {
		message := "token in environment variable TF_TOKEN_example_com is not authorized to access this module"
		r := &WorkspaceAssessmentResolver{
			assessment: &models.WorkspaceAssessment{ErrorMessage: &message},
		}

		got := r.ErrorMessage()
		require.NotNil(t, got)
		assert.Equal(t, message, *got)
	})

	t.Run("a recorded failure coexists with a previously recorded drift verdict", func(t *testing.T) {
		// The drift verdict is deliberately left standing when an attempt cannot run, so both
		// fields can be populated at once and the UI has to render that combination.
		completed := time.Now().UTC()
		r := &WorkspaceAssessmentResolver{
			assessment: &models.WorkspaceAssessment{
				HasDrift:             true,
				CompletedAtTimestamp: &completed,
				ErrorMessage:         ptr.String("module not found for source registry/example/aws"),
			},
		}

		assert.True(t, r.HasDrift())
		assert.NotNil(t, r.ErrorMessage())
		assert.NotNil(t, r.CompletedAt())
	})
}
