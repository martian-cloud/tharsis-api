package commands

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// newTestFactory builds a Factory with a real (empty) db.Client so the
// rule enforcer and plan parser the constructor wires up are non-nil, and a real
// logger so factory-built commands never carry a nil logger. The remaining
// dependencies are left as their zero values; the constructor methods under test
// only copy them through.
func newTestFactory() *Factory {
	logr, _ := logger.NewForTest()
	dbClient := &db.Client{}
	return NewFactory(logr, dbClient, nil, nil, nil, ">= 1.0.0", nil, nil)
}

func TestFactory_NewSyncJobStatus(t *testing.T) {
	f := newTestFactory()

	persistJob := func(_ context.Context) error { return nil }
	cmd := f.NewSyncJobStatus("run-1", models.JobApplyType, "job-1", models.JobFinished, persistJob)

	require.NotNil(t, cmd)
	assert.Equal(t, "run-1", cmd.RunID)
	assert.Equal(t, models.JobApplyType, cmd.JobType)
	assert.Equal(t, "job-1", cmd.JobID)
	assert.Equal(t, models.JobFinished, cmd.NewStatus)
	assert.NotNil(t, cmd.PersistJob)
}

func TestFactory_NewQueueRun(t *testing.T) {
	f := newTestFactory()

	cmd := f.NewQueueRun("run-2")

	require.NotNil(t, cmd)
	assert.Equal(t, "run-2", cmd.RunID)
	// The admitter is wired through from the factory (nil here, but the field
	// assignment path is exercised).
	assert.Equal(t, f.admitter, cmd.admitter)
}

func TestFactory_NewRun(t *testing.T) {
	f := newTestFactory()
	in := &NewRunInput{}

	cmd := f.NewRun(in)

	require.NotNil(t, cmd)
	assert.Same(t, in, cmd.in)
	assert.Equal(t, f.dbClient, cmd.dbClient)
}

func TestFactory_NewCreateDestroyRun(t *testing.T) {
	f := newTestFactory()
	in := &CreateDestroyRunInput{Subject: "subject-1", WorkspaceID: "ws-1"}

	cmd := f.NewCreateDestroyRun(in)

	require.NotNil(t, cmd)
	assert.Same(t, in, cmd.in)
	assert.Equal(t, f.dbClient, cmd.dbClient)
}

func TestFactory_NewCreateReconcileRun(t *testing.T) {
	f := newTestFactory()
	in := &CreateReconcileRunInput{Subject: "subject-1", WorkspaceID: "ws-1"}

	cmd := f.NewCreateReconcileRun(in)

	require.NotNil(t, cmd)
	assert.Same(t, in, cmd.in)
}

func TestFactory_NewCreateAssessmentRun(t *testing.T) {
	f := newTestFactory()
	version := 7
	in := &CreateAssessmentRunInput{Subject: "subject-1", WorkspaceID: "ws-1", LatestAssessmentVersion: &version}

	cmd := f.NewCreateAssessmentRun(in)

	require.NotNil(t, cmd)
	assert.Same(t, in, cmd.in)
	require.NotNil(t, cmd.in.LatestAssessmentVersion)
	assert.Equal(t, 7, *cmd.in.LatestAssessmentVersion)
}

func TestFactory_NewStartApply(t *testing.T) {
	f := newTestFactory()
	in := &StartApplyInput{RunID: "run-3"}

	cmd := f.NewStartApply(in)

	require.NotNil(t, cmd)
	assert.Same(t, in, cmd.in)
	assert.Equal(t, f.dbClient, cmd.dbClient)
	// The rule enforcer is constructed by NewFactory and threaded through.
	assert.Equal(t, f.ruleEnforcer, cmd.ruleEnforcer)
}

func TestFactory_NewCancelRun(t *testing.T) {
	f := newTestFactory()
	in := &CancelRunInput{RunID: "run-4", Force: true}

	cmd := f.NewCancelRun(in)

	require.NotNil(t, cmd)
	assert.Same(t, in, cmd.in)
	assert.Equal(t, f.dbClient, cmd.dbClient)
}

func TestFactory_NewUpdatePlanSummary(t *testing.T) {
	f := newTestFactory()
	in := &UpdatePlanSummaryInput{PlanID: "plan-1"}

	cmd := f.NewUpdatePlanSummary(in)

	require.NotNil(t, cmd)
	assert.Same(t, in, cmd.in)
	assert.Equal(t, f.dbClient, cmd.dbClient)
	// The plan parser is constructed by NewFactory.
	assert.NotNil(t, cmd.planParser)
}

func TestFactory_NewUpdatePlan(t *testing.T) {
	f := newTestFactory()
	msg := "boom"

	cmd := f.NewUpdatePlan("plan-2", true, &msg)

	require.NotNil(t, cmd)
	assert.Equal(t, "plan-2", cmd.PlanID)
	assert.True(t, cmd.HasChanges)
	require.NotNil(t, cmd.ErrorMessage)
	assert.Equal(t, "boom", *cmd.ErrorMessage)
}

func TestFactory_NewUpdateApply(t *testing.T) {
	f := newTestFactory()
	msg := "kaboom"

	cmd := f.NewUpdateApply("apply-1", &msg)

	require.NotNil(t, cmd)
	assert.Equal(t, "apply-1", cmd.ApplyID)
	require.NotNil(t, cmd.ErrorMessage)
	assert.Equal(t, "kaboom", *cmd.ErrorMessage)
}
