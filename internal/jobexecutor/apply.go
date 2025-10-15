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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

// ApplyHandler handles an apply job
type ApplyHandler struct {
	client             jobclient.Client
	cancellableCtx     context.Context
	terraformWorkspace *terraformWorkspace
	run                *types.Run
	logger             logger.Logger
	jobLogger          joblogger.Logger
	workspaceDir       string
}

// NewApplyHandler creates a new ApplyHandler
func NewApplyHandler(
	cancellableCtx context.Context,
	jobCfg *JobConfig,
	workspaceDir string,
	workspace *types.Workspace,
	run *types.Run,
	logger logger.Logger,
	jobLogger joblogger.Logger,
	client jobclient.Client,
) (*ApplyHandler, error) {
	terraformWorkspace, err := newTerraformWorkspace(cancellableCtx, jobCfg, workspaceDir, workspace, run, jobLogger, client)
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
	apply := a.run.Apply

	if a.cancellableCtx.Err() != nil {
		a.jobLogger.Errorf("Apply canceled while in progress %s", failureIcon)
		apply.Status = types.ApplyCanceled
	} else {
		a.jobLogger.Errorf("Error occurred while executing apply %s", failureIcon)
		apply.Status = types.ApplyErrored
		apply.ErrorMessage = parseTfExecError(applyErr)
	}

	// Flush all logs before updating apply state
	a.jobLogger.Flush()

	_, err := a.client.UpdateApply(ctx, apply)
	if err != nil {
		a.logger.Errorf("failed to update apply in database %v", err)
	}
}

// Execute will execute the job
func (a *ApplyHandler) Execute(ctx context.Context) error {
	apply := a.run.Apply
	if a.run.Apply == nil {
		return errors.New("cannot run apply stage because Apply is undefined")
	}

	apply.Status = types.ApplyRunning
	apply, err := a.client.UpdateApply(ctx, apply)
	if err != nil {
		return fmt.Errorf("failed to update apply %v", err)
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
		sv, csvErr := a.client.CreateStateVersion(ctx, a.run.Metadata.ID, stateFile)
		if csvErr != nil {
			return fmt.Errorf("failed to create new state version %v", csvErr)
		}
		a.jobLogger.Infof("Created new state version %s", sv.Metadata.ID)
	} else {
		a.jobLogger.Infof("No state version was created because state file is empty")
	}

	// Update apply and run status
	if a.cancellableCtx.Err() != nil || isCancellationError(cmdErr) {
		a.jobLogger.Infof("Terraform apply command gracefully exited due to job cancellation")
		apply.Status = types.ApplyCanceled
	} else if cmdErr != nil {
		a.OnError(ctx, cmdErr)
		return nil
	} else {
		apply.Status = types.ApplyFinished
	}

	// Flush all logs before updating apply state
	a.jobLogger.Flush()

	_, err = a.client.UpdateApply(ctx, apply)
	if err != nil {
		return fmt.Errorf("failed to update apply %v", err)
	}

	return nil
}

func (a *ApplyHandler) downloadPlanCache(ctx context.Context, downloadPath string) error {
	cacheFile, err := os.Create(downloadPath)
	if err != nil {
		return fmt.Errorf("failed to create temporary file for plan cache %v", err)
	}

	defer cacheFile.Close()

	return a.client.DownloadPlanCache(
		ctx,
		a.run.Plan.Metadata.ID,
		cacheFile,
	)
}
