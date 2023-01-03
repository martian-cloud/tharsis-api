package jobexecutor

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/zclconf/go-cty/cty"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

// PlanHandler handles a plan job
type PlanHandler struct {
	client             Client
	cancellableCtx     context.Context
	terraformWorkspace *terraformWorkspace
	run                *types.Run
	jobLogger          *jobLogger
	workspaceDir       string
}

// NewPlanHandler creates a new PlanHandler
func NewPlanHandler(
	cancellableCtx context.Context,
	jobCfg *JobConfig,
	workspaceDir string,
	workspace *types.Workspace,
	run *types.Run,
	jobLogger *jobLogger,
	client Client,
) *PlanHandler {
	terraformWorkspace := newTerraformWorkspace(cancellableCtx, jobCfg, workspaceDir, workspace, run, jobLogger, client)

	return &PlanHandler{
		workspaceDir:       workspaceDir,
		terraformWorkspace: terraformWorkspace,
		run:                run,
		jobLogger:          jobLogger,
		client:             client,
		cancellableCtx:     cancellableCtx,
	}
}

// OnSuccess is called after the job has been executed successfully
func (p *PlanHandler) OnSuccess(ctx context.Context) error {
	// Cleanup workspace
	return p.terraformWorkspace.close(ctx)
}

// OnError is called if the job returns an error while executing
func (p *PlanHandler) OnError(ctx context.Context, e error) error {
	// Cleanup workspace
	if err := p.terraformWorkspace.close(ctx); err != nil {
		return err
	}

	plan := p.run.Plan

	if p.cancellableCtx.Err() != nil {
		plan.Status = types.PlanCanceled
	} else {
		plan.Status = types.PlanErrored
	}
	_, err := p.client.UpdatePlan(ctx, plan)
	if err != nil {
		return fmt.Errorf("failed to update plan in database %v", err)
	}

	return nil
}

// Execute will execute the job
func (p *PlanHandler) Execute(ctx context.Context) error {
	// Get plan resource and update status to running
	plan := p.run.Plan

	plan.Status = types.PlanRunning
	plan, err := p.client.UpdatePlan(ctx, plan)
	if err != nil {
		return fmt.Errorf("failed to update plan %v", err)
	}

	tf, err := p.terraformWorkspace.init(ctx)
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "output")
	if err != nil {
		return fmt.Errorf("failed to create temp output directory %v", err)
	}
	defer os.RemoveAll(tmpDir)

	planOutputPath := fmt.Sprintf("%s/%s", tmpDir, plan.Metadata.ID)
	tfVarsFilePath, err := p.createVarsFile(ctx)
	if err != nil {
		return fmt.Errorf("failed to create tfvars file: %v", err)
	}

	hasChanges, err := tf.Plan(
		p.cancellableCtx,
		tfexec.Out(planOutputPath),
		tfexec.Destroy(p.run.IsDestroy),
		tfexec.VarFile(tfVarsFilePath),
	)

	if isCancellationError(err) {
		p.jobLogger.Infof("Terraform plan command gracefully exited due to job cancellation")
	} else if err != nil {
		return fmt.Errorf("plan operation returned an error %v", err)
	}

	tf.SetStdout(nil)
	planOutput, err := tf.ShowPlanFile(ctx, planOutputPath)
	if err != nil {
		return fmt.Errorf("failed to run show command on plan file %v", err)
	}
	tf.SetStdout(p.jobLogger)

	for _, resource := range planOutput.ResourceChanges {
		for _, action := range resource.Change.Actions {
			switch action {
			case "create":
				plan.ResourceAdditions++
			case "update":
				plan.ResourceChanges++
			case "delete":
				plan.ResourceDestructions++
			}
		}
	}

	// Upload plan cache
	planReader, err := os.Open(planOutputPath)
	if err != nil {
		return fmt.Errorf("failed to read plan output %v", err)
	}

	defer planReader.Close()

	if err = p.client.UploadPlanCache(ctx, plan, planReader); err != nil {
		return fmt.Errorf("failed to upload plan output to the object store %v", err)
	}

	p.jobLogger.Infof("\nUploaded plan output to object store\n")

	// Flush all logs before updating plan state
	p.jobLogger.Flush()

	// Update plan and run status
	if p.cancellableCtx.Err() != nil {
		plan.Status = types.PlanCanceled
	} else {
		plan.Status = types.PlanFinished
	}

	plan.HasChanges = hasChanges
	_, err = p.client.UpdatePlan(ctx, plan)
	if err != nil {
		return fmt.Errorf("failed to update plan %v", err)
	}

	return nil
}

func (p *PlanHandler) createVarsFile(ctx context.Context) (string, error) {
	// First write HCL variables
	fileContents := ""
	for _, v := range p.terraformWorkspace.variables {
		if v.Category == types.TerraformVariableCategory && v.HCL {
			fileContents += fmt.Sprintf("%s = %s\n", v.Key, *v.Value)
		}
	}

	// Parse buffer contents
	f, diag := hclwrite.ParseConfig([]byte(fileContents), "", hcl.InitialPos)
	if diag != nil {
		return "", diag
	}
	rootBody := f.Body()

	// Use hclwriter for string values to provide HCL character escaping
	for _, v := range p.terraformWorkspace.variables {
		if v.Category == types.TerraformVariableCategory && !v.HCL {
			rootBody.SetAttributeValue(v.Key, cty.StringVal(*v.Value))
		}
	}

	filePath := fmt.Sprintf("%s/run-%s.tfvars", p.workspaceDir, p.run.Metadata.ID)
	tfVarsFile, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file for tfvars %v", err)
	}

	defer tfVarsFile.Close()

	// Save file
	if _, err := f.WriteTo(tfVarsFile); err != nil {
		return "", err
	}

	return filePath, nil
}
