package resolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// TestApplyResolver_TriggeredByOnlyOnceStarted verifies an auto-apply run's apply doesn't report who
// triggered it while it is still waiting to start.
func TestApplyResolver_TriggeredByOnlyOnceStarted(t *testing.T) {
	run := &models.Run{
		Apply: &models.Apply{
			Status:      models.ApplyCreated,
			TriggeredBy: new(models.NewUserActor("user-1").WithSubject("user@example.com")),
		},
	}
	r := &ApplyResolver{run: run}

	assert.Nil(t, r.TriggeredBy())
	assert.Nil(t, r.TriggeredByActor())

	run.Apply.Status = models.ApplyPending

	triggeredBy := r.TriggeredBy()
	require.NotNil(t, triggeredBy)
	assert.Equal(t, "user@example.com", *triggeredBy)
	assert.NotNil(t, r.TriggeredByActor())
}
