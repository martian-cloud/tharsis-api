package servers

import (
	"testing"

	"github.com/stretchr/testify/assert"

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
