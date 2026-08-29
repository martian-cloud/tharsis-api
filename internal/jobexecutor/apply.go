// Package jobexecutor package
package jobexecutor

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/martian-cloud/terraform-exec/tfexec"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/jobclient"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/joblogger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/grpc/status"
)

// ApplyHandler handles an apply job
type ApplyHandler struct {
	client             jobclient.Client
	cancellableCtx     context.Context
	terraformWorkspace *terraformWorkspace
	run                *pb.Run
	logger             logger.Logger
	jobLogger          joblogger.Logger
	workspaceDir       string
}

// NewApplyHandler creates a new ApplyHandler
func NewApplyHandler(
	cancellableCtx context.Context,
	jobCfg *JobConfig,
	workspaceDir string,
	workspace *pb.Workspace,
	run *pb.Run,
	job *pb.Job,
	logger logger.Logger,
	jobLogger joblogger.Logger,
	client jobclient.Client,
) (*ApplyHandler, error) {
	terraformWorkspace, err := newTerraformWorkspace(cancellableCtx, jobCfg, workspaceDir, workspace, run, job, jobLogger, client)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize terraform workspace: %w", err)
	}

	return &ApplyHandler{
		workspaceDir:       workspaceDir,
		terraformWorkspace: terraformWorkspace,
		run:                run,
		logger:             logger,
		jobLogger:          jobLogger,
		client:             client,
		cancellableCtx:     cancellableCtx,
	}, nil
}

// Cleanup is called after the job has been executed
func (a *ApplyHandler) Cleanup(ctx context.Context) error {
	// Cleanup workspace
	return a.terraformWorkspace.close(ctx)
}

// OnError is called if the job returns an error while executing
func (a *ApplyHandler) OnError(ctx context.Context, applyErr error) {
	input := &jobclient.UpdateApplyInput{
		ID: a.run.ApplyId,
	}

	if a.cancellableCtx.Err() != nil {
		a.jobLogger.Errorf("Apply canceled while in progress %s", failureIcon)
	} else {
		a.jobLogger.Errorf("Error occurred while executing apply %s", failureIcon)
		input.ErrorMessage = parseTfExecError(applyErr)
	}

	// Flush all logs before recording the apply error. The apply node's status is
	// derived from the job status, which is reported separately.
	a.jobLogger.Flush()

	if _, err := a.client.UpdateApply(ctx, input); err != nil {
		a.logger.Errorf("failed to update apply in database %v", err)
	}
}

// Execute will execute the job
func (a *ApplyHandler) Execute(ctx context.Context) error {
	if a.run.ApplyId == "" {
		return errors.New("cannot run apply stage because Apply is undefined")
	}

	tf, err := a.terraformWorkspace.init(ctx)
	if err != nil {
		return err
	}

	stateOutputPath := fmt.Sprintf("%s/terraform-out.tfstate", a.terraformWorkspace.workspaceDir)

	tmpDir, err := os.MkdirTemp("", "downloads")
	if err != nil {
		return fmt.Errorf("failed to create temp downloads directory %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Download plan cache
	planCachePath := fmt.Sprintf("%s/plan_cache", tmpDir)
	if err = a.downloadPlanCache(ctx, planCachePath); err != nil {
		return fmt.Errorf("failed to download plan cache %v", err)
	}

	// To avoid compiler type problems, must build up a slice of PlanOptions before calling Plan
	applyOptions := []tfexec.ApplyOption{
		tfexec.DirOrPlan(planCachePath),
		tfexec.StateOut(stateOutputPath),
		tfexec.Refresh(a.run.Refresh),
		tfexec.RefreshOnly(a.run.RefreshOnly),
	}
	for _, target := range a.run.TargetAddresses {
		applyOptions = append(applyOptions, tfexec.Target(target))
	}

	// Var file can only be passed during apply stage if this terraform cli supports ephemeral inputs
	if a.terraformWorkspace.capabilities.ephemeralInputs {
		tfVarsFilePath, _, err := a.terraformWorkspace.createVarsFile()
		if err != nil {
			return fmt.Errorf("failed to process variables: %v", err)
		}
		applyOptions = append(applyOptions, tfexec.VarFile(tfVarsFilePath))
	}

	// Run Apply Cmd
	cmdErr := tf.Apply(a.cancellableCtx, applyOptions...)

	stateFile, err := os.Open(stateOutputPath) // nosemgrep: gosec.G304-1
	if err != nil {
		return fmt.Errorf("failed to read state output %v", err)
	}

	defer stateFile.Close()

	// Check if state file exists
	stateFileStats, err := os.Stat(stateOutputPath)

	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to find state file %v", err)
	} else if stateFileStats.Size() > 0 {
		// Create new state version
		sv, csvErr := a.client.CreateStateVersion(ctx, a.run.Metadata.Id, stateFile)
		if csvErr != nil {
			if isOutputLimitViolation(csvErr) {
				// The state version was created successfully server-side; only some
				// outputs were rejected for exceeding configured size/count limits.
				// The full state is safely persisted (nothing to recover), so skip
				// the raw-state dump below and report accurately that the state
				// version exists rather than implying its creation failed. The job
				// still ends in error: not every output was stored, which can break
				// other workspaces that depend on them via remote state.
				return fmt.Errorf("state version was created, but some outputs were not saved: %s", status.Convert(csvErr).Message())
			}

			// Log the raw state data so it can be recovered from job logs if needed.
			if stateData, readErr := os.ReadFile(stateOutputPath); readErr == nil {
				a.jobLogger.Errorf("State data for recovery (run %s):\n%s", a.run.Metadata.Id, string(stateData))
			}
			return fmt.Errorf("failed to create new state version %v", csvErr)
		}
		a.jobLogger.Infof("Created new state version %s", sv.Metadata.Id)

		// Store the "terraform show -json" rendering of the state alongside the state itself. This is
		// the representation external consumers read (policy evaluation, integrations); the raw file is
		// Terraform's internal serialization. Rendering has to happen here because it needs the provider
		// schemas to decode each resource's attributes, and this working directory already has them.
		a.uploadStateVersionJSON(ctx, tf, sv.Metadata.Id, stateOutputPath)
	} else {
		a.jobLogger.Infof("No state version was created because state file is empty")
	}

	// The apply node's status is derived from the job status (reported separately),
	// so the runner only needs to surface an execution error here.
	if a.cancellableCtx.Err() != nil || isCancellationError(cmdErr) {
		a.jobLogger.Infof("Terraform apply command gracefully exited due to job cancellation")
	} else if cmdErr != nil {
		// Return the error so the executor marks the job (and thus the apply node)
		// failed; it also invokes OnError to record the error message. Returning
		// nil here would let the job finish successfully despite the apply failing.
		// The state version is still created above so partial applies are recorded.
		return cmdErr
	}

	a.jobLogger.Flush()
	return nil
}

// uploadStateVersionJSON renders the state file that the apply just wrote into the documented
// "terraform show -json" state representation and stores it against the state version. Every failure
// is logged and swallowed: the state itself is already persisted, so the job's outcome must not hinge
// on this secondary artifact.
func (a *ApplyHandler) uploadStateVersionJSON(ctx context.Context, tf *tfexec.Terraform, stateVersionID, statePath string) {
	// Detach stdout while rendering so the state does not land in the job log, then restore it for any
	// remaining commands. This mirrors how the plan stage brackets ShowPlanFile.
	tf.SetStdout(nil)
	defer tf.SetStdout(a.jobLogger)

	stateJSON, err := tf.ShowStateFile(ctx, statePath)
	if err != nil {
		a.jobLogger.Errorf("failed to render state file as JSON; policies and integrations will not see a state representation for this run: %v", err)
		return
	}

	if err := a.client.UploadStateVersionJSON(ctx, stateVersionID, stateJSON); err != nil {
		a.jobLogger.Errorf("failed to upload state version JSON: %v", err)
	}
}

func (a *ApplyHandler) downloadPlanCache(ctx context.Context, downloadPath string) error {
	cacheFile, err := os.Create(downloadPath)
	if err != nil {
		return fmt.Errorf("failed to create temporary file for plan cache %v", err)
	}

	defer cacheFile.Close()

	return a.client.DownloadPlanCache(
		ctx,
		a.run.PlanId,
		cacheFile,
	)
}
