package jobexecutor

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/smithy-go/ptr"
	"github.com/hashicorp/go-slug"
	"github.com/hashicorp/go-version"
	hcInstall "github.com/hashicorp/hc-install"
	"github.com/hashicorp/hc-install/fs"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/src"
	"github.com/hashicorp/terraform-exec/tfexec"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/http"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/module"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

const (
	backendOverride = `
terraform {
	backend "local" {}
}
`
	// Ensure binary not found error.
	hcInstallBinaryNotFoundErr = "unable to find, install"
)

type terraformWorkspace struct {
	cliDownloader     cliDownloader
	cancellableCtx    context.Context
	client            Client
	jobCfg            *JobConfig
	workspace         *types.Workspace
	run               *types.Run
	jobLogger         *jobLogger
	managedIdentities *managedIdentities
	fullEnv           map[string]string
	workspaceDir      string
	variables         []types.RunVariable
	pathsToRemove     []string
}

func newTerraformWorkspace(
	cancellableCtx context.Context,
	jobCfg *JobConfig,
	workspaceDir string,
	workspace *types.Workspace,
	run *types.Run,
	jobLogger *jobLogger,
	client Client,
) *terraformWorkspace {
	managedIdentities := newManagedIdentities(
		workspace.Metadata.ID,
		workspaceDir,
		jobLogger,
		client,
	)

	return &terraformWorkspace{
		cancellableCtx:    cancellableCtx,
		jobCfg:            jobCfg,
		workspaceDir:      workspaceDir,
		workspace:         workspace,
		run:               run,
		jobLogger:         jobLogger,
		client:            client,
		managedIdentities: managedIdentities,
		fullEnv:           map[string]string{},
		cliDownloader: *newCLIDownloader(
			http.NewHTTPClient(),
			client,
		),
	}
}

func (t *terraformWorkspace) close(ctx context.Context) error {
	// Remove temporary files and directories.
	for _, toRemove := range t.pathsToRemove {
		os.RemoveAll(toRemove)
	}

	// Cleanup managed identity resources
	return t.managedIdentities.close(ctx)
}

// init prepares for and does "terraform init".
func (t *terraformWorkspace) init(ctx context.Context) (*tfexec.Terraform, error) {
	// Get run variables
	variables, err := t.client.GetRunVariables(ctx, t.run.Metadata.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get run variables %v", err)
	}
	t.jobLogger.Infof("Resolved run variables")

	// Add run variables to environment
	for _, v := range variables {
		if v.Category == types.EnvironmentVariableCategory {
			t.fullEnv[v.Key] = *v.Value
		}
	}

	// Add built-in variables to environment
	if envErr := t.setBuiltInEnvVars(ctx); envErr != nil {
		return nil, envErr
	}

	t.variables = variables

	managedIdentityEnv, err := t.managedIdentities.initialize(ctx)
	if err != nil {
		return nil, err
	}

	for k, v := range managedIdentityEnv {
		t.fullEnv[k] = v
	}

	// Handle a possible configuration version.  Configuration version and module
	// source are mutually exclusive, so downloading to workspaceDir is okay.
	if t.run.ConfigurationVersionID != nil {

		t.jobLogger.Infof("Downloading configuration version %s \n", *t.run.ConfigurationVersionID)

		if err = t.downloadConfigurationVersion(ctx); err != nil {
			return nil, err
		}

	}

	// Handle a possible module source (and maybe version).
	var resolvedModuleSource *string
	if t.run.ModuleSource != nil {
		if t.run.ModuleVersion != nil {
			// Registry-style module source; version is always defined in this case
			t.jobLogger.Infof("Resolving module version %s/%s", *t.run.ModuleSource, *t.run.ModuleVersion)

			presignedURL, rErr := resolveModuleSource(*t.run.ModuleSource, *t.run.ModuleVersion, t.fullEnv)
			if rErr != nil {
				return nil, fmt.Errorf("failed to resolve module source: %s", rErr)
			}

			if t.run.ModuleDigest != nil {
				// Add required checksum if module digest is defined. The go-getter library will verify the checksum when
				// downloading the module
				t.jobLogger.Infof("Module digest: %s", *t.run.ModuleDigest)
				resolvedModuleSource = ptr.String(fmt.Sprintf("%s&checksum=sha256:%s", presignedURL, *t.run.ModuleDigest))
			} else {
				resolvedModuleSource = &presignedURL
			}
		} else {
			// Non-registry-style module source.
			resolvedModuleSource = t.run.ModuleSource
		}
	}

	// Must pass PATH to fullEnv or Terraform fails when it attempts
	// to find files in the user's home directory.
	if val := os.Getenv("PATH"); val != "" {
		t.fullEnv["PATH"] = val
	}

	// Convert Terraform version to hcInstall equivalent.
	version, err := version.NewVersion(t.run.TerraformVersion)
	if err != nil {
		return nil, err
	}

	execPath, err := hcInstall.NewInstaller().Ensure(ctx, []src.Source{&fs.ExactVersion{
		Product: product.Terraform,
		Version: version,
	}})
	if err != nil {
		// Ensure command returns a custom error,
		// must test it using a prefix of the error.
		if !strings.HasPrefix(err.Error(), hcInstallBinaryNotFoundErr) {
			return nil, fmt.Errorf("failed to find a Terraform executable: %v", err)
		}

		t.jobLogger.Infof("Downloading Terraform CLI version %s", t.run.TerraformVersion)

		// Since the binary does not exist, create it.
		execPath, err = t.cliDownloader.Download(ctx, t.run.TerraformVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to download Terraform CLI version %s: %v", t.run.TerraformVersion, err)
		}

		// Contains the temporary directory that needs to be removed
		// after the job completes.
		t.pathsToRemove = append(t.pathsToRemove, filepath.Dir(execPath))
	}

	tf, err := tfexec.NewTerraform(t.workspaceDir, execPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tfexec: %v", err)
	}

	if err = t.mapSupportedTerraformEnvConfiguration(tf); err != nil {
		return nil, err
	}

	// Set environment variables
	if err = tf.SetEnv(tfexec.CleanEnv(t.fullEnv)); err != nil {
		return nil, fmt.Errorf("failed to set environment variables: %v", err)
	}

	tf.SetLogger(log.Default())

	// If we're doing a module source, we have to do two init operations.
	// The first init operation uses FromModule and backend=false to download the module.
	// Then, the override file is written.
	// The second init operation has no init options.
	// If we're doing a configuration version, the first init operation is skipped.

	if resolvedModuleSource != nil {
		err = tf.Init(t.cancellableCtx, // Use cancellable context here so that init command can be manually cancelled
			tfexec.FromModule(*resolvedModuleSource), // Add a -from-module init option.  (Don't try to url.QueryEscape it.)
			tfexec.Backend(false),                    // Nullify any backend in the original module.
		)
		if isCancellationError(err) {
			return nil, fmt.Errorf("job cancelled while first terraform init command was in progress")
		} else if err != nil {
			if strings.Contains(err.Error(), "Checksums did not match") && t.run.ModuleDigest != nil {
				return nil, fmt.Errorf(
					"failed to download root module: checksum did not match expected value of %s",
					*t.run.ModuleDigest,
				)
			}
			return nil, fmt.Errorf("failed to download root module %v", err)
		}
	}

	// These output redirections must be done _AFTER_ the above init operation
	// (which is done only for module source jobs) or Terraform logs the full
	// final URL that contains a valid (even if temporary) token.
	tf.SetStdout(t.jobLogger)
	tf.SetStderr(t.jobLogger)

	// Write an override file to set backend to local.
	overridePath := filepath.Join(t.workspaceDir, "override.tf")
	if err = os.WriteFile(overridePath, []byte(backendOverride), 0o600); err != nil {
		return nil, fmt.Errorf("failed to write terraform override file %v", err)
	}

	// If a current workspace state exists, download it to workspaceDir.
	// This must happen after the first init or remote module source download refuses to try.
	if t.workspace.CurrentStateVersion != nil {
		if err = t.downloadCurrentStateVersion(ctx); err != nil {
			return nil, err
		}
	}

	// Use cancellable context here so that init command can be manually cancelled
	err = tf.Init(t.cancellableCtx)
	if isCancellationError(err) {
		return nil, fmt.Errorf("job cancelled while terraform init command was in progress")
	} else if err != nil {
		return nil, fmt.Errorf("failed to run init command %v", err)
	}

	return tf, nil
}

// mapSupportedTerraformEnvConfiguration
func (t *terraformWorkspace) mapSupportedTerraformEnvConfiguration(tf *tfexec.Terraform) error {
	var (
		supportedTFSettings = map[string]func(string) error{
			"TF_LOG":          tf.SetLog,
			"TF_LOG_CORE":     tf.SetLogCore,
			"TF_LOG_PROVIDER": tf.SetLogProvider,
		}

		logPathSet = false
	)

	// Instead of just iterating over the map and applying the setting,
	// we should report to the user when they use an unsupported environment
	// variable for configuration.
	for _, env := range tfexec.ProhibitedEnv(t.fullEnv) {
		fn, ok := supportedTFSettings[env]
		if !ok {
			t.jobLogger.Errorf("Environment variable %s is not currently supported by the job executor", env)
			continue
		}

		// In case we enable additional settings, check for TF_LOG
		if strings.HasPrefix(env, "TF_LOG") && !logPathSet {
			if err := tf.SetLogPath(os.Stderr.Name()); err != nil {
				return fmt.Errorf("failed to set log path: %w", err)
			}
			logPathSet = true
		}

		if err := fn(t.fullEnv[env]); err != nil {
			// return on first instance of failure
			return fmt.Errorf("failed to set %s: %w", env, err)
		}
	}

	return nil
}

func (t *terraformWorkspace) downloadConfigurationVersion(ctx context.Context) error {
	tmpDownloadDir, err := os.MkdirTemp("", "downloads")
	if err != nil {
		return fmt.Errorf("failed to create temp downloads directory %v", err)
	}
	defer os.RemoveAll(tmpDownloadDir)

	cvFilePath := fmt.Sprintf("%s/%s.tar.gz", tmpDownloadDir, *t.run.ConfigurationVersionID)

	cvFile, err := os.Create(cvFilePath)
	if err != nil {
		return fmt.Errorf(
			"failed to create temporary configuration version file for download: %v", err,
		)
	}

	defer cvFile.Close()

	cv, err := t.client.GetConfigurationVersion(ctx, *t.run.ConfigurationVersionID)
	if err != nil {
		return fmt.Errorf(
			"failed to query configuration version from database: %v",
			err,
		)
	}

	if err := t.client.DownloadConfigurationVersion(ctx, cv, cvFile); err != nil {
		return err
	}

	// Rewind file to start
	if _, err := cvFile.Seek(0, io.SeekStart); err != nil {
		return err
	}

	return slug.Unpack(cvFile, t.workspaceDir)
}

func (t *terraformWorkspace) downloadCurrentStateVersion(ctx context.Context) error {
	stateVersion := t.workspace.CurrentStateVersion

	stateFile, err := os.Create(filepath.Join(t.workspaceDir, "terraform.tfstate"))
	if err != nil {
		return fmt.Errorf(
			"failed to create temporary file for current terraform state: %v",
			err,
		)
	}

	defer stateFile.Close()

	return t.client.DownloadStateVersion(ctx, stateVersion, stateFile)
}

// setBuiltInEnvVars will add Tharsis built in environment variables for the job.
func (t *terraformWorkspace) setBuiltInEnvVars(ctx context.Context) error {
	// Set THARSIS_GROUP_PATH
	t.fullEnv["THARSIS_GROUP_PATH"] = t.workspace.FullPath[:strings.LastIndex(t.workspace.FullPath, "/")]

	apiURL, err := url.Parse(t.jobCfg.APIEndpoint)
	if err != nil {
		return fmt.Errorf("failed to parse API URL %v", err)
	}

	// Set TF_TOKEN_<host>
	apiEncHost, err := module.BuildTokenEnvVar(apiURL.Host)
	if err != nil {
		return fmt.Errorf("failed to encode API URL for environment variable: %v", err)
	}
	t.fullEnv[apiEncHost] = t.jobCfg.JobToken

	if t.jobCfg.DiscoveryProtocolHost != "" {
		if dpEncHost, err := module.BuildTokenEnvVar(t.jobCfg.DiscoveryProtocolHost); err == nil {
			t.fullEnv[dpEncHost] = t.jobCfg.JobToken
		} else {
			t.jobLogger.logger.Infof("failed to encode Discovery Protocol Host: %v", err)
		}
	}

	// Set THARSIS_ENDPOINT which is used by the Terraform Tharsis Provider
	t.fullEnv["THARSIS_ENDPOINT"] = t.jobCfg.APIEndpoint

	// Placeholder in case we need to make an API call to get additional variables
	_ = ctx

	return nil
}
