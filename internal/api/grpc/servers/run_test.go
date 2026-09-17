package servers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
)

// TestToPBRunStatus_PostApplyRunning verifies the new status maps to its protobuf counterpart.
func TestToPBRunStatus_PostApplyRunning(t *testing.T) {
	assert.Equal(t, pb.RunStatus_POST_APPLY_RUNNING, toPBRunStatus(models.RunPostApplyRunning))
}

// TestToPBRunStatus_TransientCompletedStatusesFallBackToUnspecified verifies the two statuses
// deliberately absent from the wire enum (see the RunStatus enum comment in run.proto) fall back to
// UNSPECIFIED via toPBRunStatus's documented behavior for an unmapped name, rather than the lookup
// panicking or the caller mistaking a raw string for a value the wire enum understands. This is
// safe only because no persisted run is ever observed in either status (enforced by
// TestStateMachine_NoRunEverRestsOnTransientCompletedStatuses in the statemachine package).
func TestToPBRunStatus_TransientCompletedStatusesFallBackToUnspecified(t *testing.T) {
	for _, status := range []models.RunStatus{models.RunPostPlanCompleted, models.RunPostApplyCompleted} {
		assert.Equalf(t, pb.RunStatus_UNSPECIFIED, toPBRunStatus(status), "%q should fall back to UNSPECIFIED", status)
	}
}

// TestToPBRunStatus_EveryOtherStatusIsMapped verifies every run status other than the two
// deliberately-transient ones above has an explicit, non-UNSPECIFIED protobuf counterpart, so
// AllRunStatuses stays the single source of truth for "is this status wired all the way to the
// runner" without a status silently degrading to UNSPECIFIED by omission.
func TestToPBRunStatus_EveryOtherStatusIsMapped(t *testing.T) {
	transient := map[models.RunStatus]bool{
		models.RunPostPlanCompleted:  true,
		models.RunPostApplyCompleted: true,
	}
	for _, status := range models.AllRunStatuses {
		if transient[status] {
			continue
		}
		assert.NotEqualf(t, pb.RunStatus_UNSPECIFIED, toPBRunStatus(status), "%q has no protobuf mapping", status)
	}
}

// TestRunAnnotations_ProtoRoundTrip verifies the annotation converters preserve every field
// (including the optional link and duplicate keys, in order) when going model -> proto -> model.
func TestRunAnnotations_ProtoRoundTrip(t *testing.T) {
	link := "https://gitlab.example.com/x/-/commit/a1b2c3d4"
	original := []*models.RunAnnotation{
		{Key: "commit", Value: "a1b2c3d4", Link: &link}, // with link
		{Key: "ref", Value: "main"},                     // without link
		{Key: "commit", Value: "e5f6g7h8"},              // duplicate key, order preserved
	}

	pbAnnotations := toPBRunAnnotations(original)
	require.Len(t, pbAnnotations, 3)
	assert.Equal(t, "commit", pbAnnotations[0].Key)
	assert.Equal(t, "a1b2c3d4", pbAnnotations[0].Value)
	require.NotNil(t, pbAnnotations[0].Link)
	assert.Equal(t, link, *pbAnnotations[0].Link)
	assert.Nil(t, pbAnnotations[1].Link)

	roundTripped := fromPBRunAnnotations(pbAnnotations)
	require.Len(t, roundTripped, 3)
	for i := range original {
		assert.Equal(t, original[i].Key, roundTripped[i].Key)
		assert.Equal(t, original[i].Value, roundTripped[i].Value)
		if original[i].Link == nil {
			assert.Nil(t, roundTripped[i].Link)
		} else {
			require.NotNil(t, roundTripped[i].Link)
			assert.Equal(t, *original[i].Link, *roundTripped[i].Link)
		}
	}
}

// TestRunAnnotations_EmptyAndNil verifies both converters map empty/nil input to nil, matching how
// run creation treats "no annotations".
func TestRunAnnotations_EmptyAndNil(t *testing.T) {
	assert.Nil(t, toPBRunAnnotations(nil))
	assert.Nil(t, toPBRunAnnotations([]*models.RunAnnotation{}))
	assert.Nil(t, fromPBRunAnnotations(nil))
	assert.Nil(t, fromPBRunAnnotations([]*pb.RunAnnotation{}))
}
