package commands

import (
	"context"
	"io"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// UploadPlanBinaryInput carries everything UploadPlanBinary needs across its
// Prepare and Execute phases.
type UploadPlanBinaryInput struct {
	Body   io.Reader
	PlanID string
}

// UploadPlanBinary stores the plan binary in object storage in Prepare (once, before the
// transaction), then in Execute records the object key on the plan node. Uploading in Prepare keeps
// the S3 write out of the transaction and off the OLE-retry path: the executor uploads the plan
// binary and the plan data at the same time, and both update the same run, so this command has to
// tolerate losing the version race without repeating the upload.
type UploadPlanBinary struct {
	dbClient      *db.Client
	artifactStore workspace.ArtifactStore
	in            *UploadPlanBinaryInput

	// Populated by Prepare.
	retainFn db.RetainObjectRefFunc
	runID    string
	cacheKey string
}

// Prepare resolves the run and uploads the plan binary. It runs before the transaction is opened.
func (c *UploadPlanBinary) Prepare(ctx context.Context) error {
	run, err := c.dbClient.Runs.GetRunByNodeID(ctx, c.in.PlanID)
	if err != nil {
		return errors.Wrap(err, "failed to get run by plan ID")
	}
	if run == nil {
		return errors.New("plan with ID %s not found", c.in.PlanID, errors.WithErrorCode(errors.ENotFound))
	}

	retainFn, cacheKey, err := c.artifactStore.UploadPlanCache(ctx, run, c.in.Body)
	if err != nil {
		return errors.Wrap(err, "failed to upload plan cache")
	}

	c.runID = run.Metadata.ID
	c.cacheKey = cacheKey
	c.retainFn = retainFn
	return nil
}

// Execute records the uploaded object key on the plan node and links the stored object to the run.
func (c *UploadPlanBinary) Execute(ctx context.Context, input *types.ExecuteInput) error {
	run, err := input.RunStore.GetRunByID(ctx, c.runID)
	if err != nil {
		return err
	}

	run.Plan.CacheObjectStoreKey = &c.cacheKey

	if err = c.retainFn(ctx, c.runID); err != nil {
		return errors.Wrap(err, "failed to link plan cache object store ref")
	}

	return nil
}
