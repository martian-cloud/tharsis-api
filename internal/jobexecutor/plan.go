package jobexecutor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/martian-cloud/terraform-exec/tfexec"
	"github.com/zclconf/go-cty/cty"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/jobclient"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/joblogger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
	"golang.org/x/sync/errgroup"
)

// PlanHandler handles a plan job
type PlanHandler struct {
	client             jobclient.Client
	cancellableCtx     context.Context
	terraformWorkspace *terraformWorkspace
	run                *types.Run
	logger             logger.Logger
	jobLogger          joblogger.Logger
	workspaceDir       string
}

// NewPlanHandler creates a new PlanHandler
func NewPlanHandler(
	cancellableCtx context.Context,
	jobCfg *JobConfig,
	workspaceDir string,
	workspace *types.Workspace,
	run *types.Run,
	logger logger.Logger,
	jobLogger joblogger.Logger,
	client jobclient.Client,
) *PlanHandler {
	terraformWorkspace := newTerraformWorkspace(cancellableCtx, jobCfg, workspaceDir, workspace, run, jobLogger, client)

	return &PlanHandler{
		workspaceDir:       workspaceDir,
		terraformWorkspace: terraformWorkspace,
		run:                run,
		logger:             logger,
		jobLogger:          jobLogger,
		client:             client,
		cancellableCtx:     cancellableCtx,
	}
}

// Cleanup is called after the job has been executed
func (p *PlanHandler) Cleanup(ctx context.Context) error {
	// Cleanup workspace
	return p.terraformWorkspace.close(ctx)
}

// OnError is called if the job returns an error while executing
func (p *PlanHandler) OnError(ctx context.Context, planErr error) {
	plan := p.run.Plan

	if p.cancellableCtx.Err() != nil {
		p.jobLogger.Errorf("Plan canceled while in progress %s", failureIcon)
		plan.Status = types.PlanCanceled
	} else {
		p.jobLogger.Errorf("Error occurred while executing plan %s", failureIcon)
		plan.Status = types.PlanErrored
		plan.ErrorMessage = parseTfExecError(planErr)
	}

	// Flush all logs before updating apply state
	p.jobLogger.Flush()

	_, err := p.client.UpdatePlan(ctx, plan)
	if err != nil {
		p.logger.Errorf("failed to update plan in database %v", err)
	}
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

	// Parse terraform config
	terraformModule, diag := tfconfig.LoadModule(p.workspaceDir)
	if diag.HasErrors() {
		return fmt.Errorf("failed to load terraform module %v", diag)
	}

	tfVarsFilePath, variablesIncludedInTFConfig, err := p.createVarsFile(terraformModule)
	if err != nil {
		return fmt.Errorf("failed to process variables: %v", err)
	}

	if len(variablesIncludedInTFConfig) > 0 {
		// Update run variables with latest usage.
		if err = p.client.SetVariablesIncludedInTFConfig(ctx, p.run.Metadata.ID, variablesIncludedInTFConfig); err != nil {
			return fmt.Errorf("failed to set variables included in tfconfig: %v", err)
		}
	}

	planOutputPath := fmt.Sprintf("%s/%s", tmpDir, plan.Metadata.ID)

	// To avoid compiler type problems, must build up a slice of PlanOptions before calling Plan
	planOptions := []tfexec.PlanOption{
		tfexec.Out(planOutputPath),
		tfexec.Destroy(p.run.IsDestroy),
		tfexec.VarFile(tfVarsFilePath),
		tfexec.Refresh(p.run.Refresh),
		tfexec.RefreshOnly(p.run.RefreshOnly),
	}
	for _, target := range p.run.TargetAddresses {
		planOptions = append(planOptions, tfexec.Target(target))
	}

	// Run Plan Cmd
	hasChanges, err := tf.Plan(p.cancellableCtx, planOptions...)

	if isCancellationError(err) {
		p.jobLogger.Infof("Terraform plan command gracefully exited due to job cancellation")
	} else if err != nil {
		p.OnError(ctx, err)
		return nil
	}

	_, err = p.jobLogger.Write([]byte("\nPreparing plan output...\n"))
	if err != nil {
		return err
	}

	tf.SetStdout(nil)

	// Get plan
	planJSON, err := tf.ShowPlanFile(ctx, planOutputPath)
	if err != nil {
		return fmt.Errorf("failed to run show command on plan file %v", err)
	}

	// Provider schemas
	providerSchemasJSON, err := tf.ProvidersSchema(ctx)
	if err != nil {
		return fmt.Errorf("failed to run provider schema command: %v", err)
	}

	tf.SetStdout(p.jobLogger)

	// Use error group to upload plan cache and plan data in parallel
	eg := errgroup.Group{}

	eg.Go(func() error {
		// Upload plan cache
		planReader, err := os.Open(planOutputPath) // nosemgrep: gosec.G304-1
		if err != nil {
			return fmt.Errorf("failed to read plan output %v", err)
		}

		defer planReader.Close()

		if err = p.client.UploadPlanCache(ctx, plan, planReader); err != nil {
			return fmt.Errorf("failed to upload plan binary%v", err)
		}
		return nil
	})

	eg.Go(func() error {
		if err = p.client.UploadPlanData(ctx, plan, planJSON, providerSchemasJSON); err != nil {
			// Log error and continue
			p.jobLogger.Errorf("failed to upload plan json output %v", err)
		}
		return nil
	})

	// Wait for both uploads to finish
	if err = eg.Wait(); err != nil {
		return err
	}

	// Update plan and run status
	if p.cancellableCtx.Err() != nil {
		plan.Status = types.PlanCanceled
		p.jobLogger.Write([]byte("\n"))
		p.jobLogger.Infof("Plan was canceled %s", failureIcon)
	} else {
		plan.Status = types.PlanFinished
		p.jobLogger.Write([]byte("\n"))
		p.jobLogger.Infof("Plan complete! %s", successIcon)
	}

	// Flush all logs before updating plan
	p.jobLogger.Flush()

	plan.HasChanges = hasChanges
	_, err = p.client.UpdatePlan(ctx, plan)
	if err != nil {
		return fmt.Errorf("failed to update plan %v", err)
	}

	return nil
}

func (p *PlanHandler) createVarsFile(terraformModule *tfconfig.Module) (string, []string, error) {
	// Get all variables in the module
	hclVariables := []types.RunVariable{}
	stringVariables := []types.RunVariable{}
	variablesIncludedInTFConfig := []string{}

	for _, v := range p.terraformWorkspace.variables {
		if v.Category != types.TerraformVariableCategory {
			continue
		}

		// Check if there is an hcl definition for this variable
		variable, ok := terraformModule.Variables[v.Key]
		if !ok {
			// Make it easier for the user to identity where the variable is coming from.
			if v.NamespacePath != nil && *v.NamespacePath == p.terraformWorkspace.workspace.FullPath {
				p.jobLogger.Warningf("WARNING: Workspace variable %q has a value but is not defined in the terraform module.", v.Key)
			}

			if v.NamespacePath == nil {
				p.jobLogger.Warningf("WARNING: Run variable %q has a value but is not defined in the terraform module.", v.Key)
			}

			continue
		}

		// Verify that the variable definition is marked as sensitive
		if v.Sensitive && !variable.Sensitive {
			return "", nil, fmt.Errorf(
				"variable %q is marked as sensitive but the hcl definition in the terraform file %q is not sensitive, sensitive variables can only be passed to variable definitions with sensitive set to true",
				v.Key,
				filepath.Base(variable.Pos.Filename),
			)
		}

		variablesIncludedInTFConfig = append(variablesIncludedInTFConfig, v.Key)

		if isHCLVariable(v.Value, variable) {
			hclVariables = append(hclVariables, v)
		} else {
			stringVariables = append(stringVariables, v)
		}
	}

	// First write HCL variables
	fileContents := ""
	for _, v := range hclVariables {
		fileContents += fmt.Sprintf("%s = %s\n", v.Key, *v.Value)
	}

	// Parse buffer contents
	f, diag := hclwrite.ParseConfig([]byte(fileContents), "", hcl.InitialPos)
	if diag != nil {
		return "", nil, diag
	}
	rootBody := f.Body()

	// Use hclwriter for string values to provide HCL character escaping
	for _, v := range stringVariables {
		rootBody.SetAttributeValue(v.Key, cty.StringVal(*v.Value))
	}

	filePath := fmt.Sprintf("%s/run-%s.tfvars", p.workspaceDir, p.run.Metadata.ID)
	tfVarsFile, err := os.Create(filePath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temporary file for tfvars %v", err)
	}

	defer tfVarsFile.Close()

	// Save file
	if _, err := f.WriteTo(tfVarsFile); err != nil {
		return "", nil, err
	}

	return filePath, variablesIncludedInTFConfig, nil
}
