package servers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
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

// TestJobServer_MarkJobsCleanedUpInvalidJobID verifies a job ID that is not a GID is rejected as invalid
// input instead of reaching the job service as a garbage ID.
func TestJobServer_MarkJobsCleanedUpInvalidJobID(t *testing.T) {
	// The request is rejected before the job service is reached, so the catalog has none.
	catalog := &services.Catalog{}
	catalog.Init()
	ctx := auth.WithCaller(context.Background(), &auth.SystemCaller{})

	_, err := NewJobServer(catalog).MarkJobsCleanedUp(ctx, &pb.MarkJobsCleanedUpRequest{
		RunnerId: gid.ToGlobalID(types.RunnerModelType, testGroupID),
		JobIds:   []string{gid.ToGlobalID(types.JobModelType, testWorkspaceID), "not-a-gid"},
	})
	require.Error(t, err)
	assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
}
