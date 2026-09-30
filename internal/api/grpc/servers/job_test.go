package servers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// TestToPBJob_DispatcherData verifies dispatcher data is carried through to the protobuf
// job (and that a nil map maps to a nil field, matching a never-dispatched job).
func TestToPBJob_DispatcherData(t *testing.T) {
	now := time.Now()
	meta := models.ResourceMetadata{CreationTimestamp: &now, LastUpdatedTimestamp: &now}

	withData := &models.Job{Metadata: meta, DispatcherData: map[string]string{"podName": "tharsis-job-abc"}}
	assert.Equal(t, map[string]string{"podName": "tharsis-job-abc"}, toPBJob(withData).DispatcherData)

	assert.Nil(t, toPBJob(&models.Job{Metadata: meta}).DispatcherData)
}
