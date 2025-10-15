package jobexecutor

import (
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/aws/smithy-go/ptr"
	"github.com/hashicorp/go-slug"
	"github.com/hashicorp/go-version"
	hcInstall "github.com/hashicorp/hc-install"
	"github.com/hashicorp/hc-install/fs"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/src"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/martian-cloud/terraform-exec/tfexec"
	"github.com/zclconf/go-cty/cty"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/http"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/jobclient"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/joblogger"
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

	// Env variable name prefix used by Terraform for tokens. Format is TF_TOKEN_<host>
	tfTokenVarPrefix = "TF_TOKEN_"

	// Terraform variable suffix for write-only version variables that are automatically injected
	writeOnlyVariableSuffix = "_wo_version"

	// Ephemeral input minimum version
	ephemeralInputCapabilityMinimumTerraformVersion = "1.11"
)

type capabilities struct {
	ephemeralInputs bool
}

func newCapabilities(terraformVersion *version.Version) (*capabilities, error) {
	minimumEphemeralInputsVersion, err := version.NewVersion(ephemeralInputCapabilityMinimumTerraformVersion)
	if err != nil {
		return nil, err
	}

	return &capabilities{
		ephemeralInputs: terraformVersion.GreaterThanOrEqual(minimumEphemeralInputsVersion),
	}, nil
}

type terraformWorkspace struct {
	cliDownloader     cliDownloader
	cancellableCtx    context.Context
	client            jobclient.Client
	jobCfg            *JobConfig
	workspace         *types.Workspace
	run               *types.Run
	jobLogger         joblogger.Logger
	managedIdentities *managedIdentities
	fullEnv           map[string]string
	workspaceDir      string
	variables         []types.RunVariable
	pathsToRemove     []string
	credentialHelper  *credentialHelper
	capabilities      *capabilities
	terraformVersion  *version.Version
}

func newTerraformWorkspace(
	cancellableCtx context.Context,
	jobCfg *JobConfig,
	workspaceDir string,
	workspace *types.Workspace,
	run *types.Run,
	jobLogger joblogger.Logger,
	client jobclient.Client,
) (*terraformWorkspace, error) {
	managedIdentities := newManagedIdentities(
		workspace.Metadata.ID,
		jobLogger,
		client,
		jobCfg,
	)

	terraformVersion, err := version.NewVersion(run.TerraformVersion)
	if err != nil {
		return nil, err
	}

	capabilities, err := newCapabilities(terraformVersion)
	if err != nil {
		return nil, err
	}

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
		credentialHelper: newCredentialHelper(),
		capabilities:     capabilities,
		terraformVersion: terraformVersion,
	}, nil
}

func (t *terraformWorkspace) close(ctx context.Context) error {
	// Remove temporary files and directories.
	for _, toRemove := range t.pathsToRemove {
		os.RemoveAll(toRemove)
	}

	t.credentialHelper.close()

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

	t.variables = variables

	managedIdentitiesResponse, err := t.managedIdentities.initialize(ctx)
	if err != nil {
		return nil, err
	}

	for k, v := range managedIdentitiesResponse.Env {
		t.fullEnv[k] = v
	}

	// Add built-in variables to environment
	if envErr := t.setBuiltInEnvVars(ctx, managedIdentitiesResponse.HostCredentialFileMapping); envErr != nil {
		return nil, envErr
	}

	// Handle a possible configuration version.  Configuration version and module
	// source are mutually exclusive, so downloading to workspaceDir is okay.
	if t.run.ConfigurationVersionID != nil {

		t.jobLogger.Infof("Downloading configuration version %s", *t.run.ConfigurationVersionID)

		if err = t.downloadConfigurationVersion(ctx); err != nil {
			return nil, err
		}
	}

	// If the above set any tokens for a federated registry remote host, potentially replace the tokens.
	federatedRegistryTokens, err := t.client.CreateFederatedRegistryTokens(ctx, &types.CreateFederatedRegistryTokensInput{
		JobID: t.jobCfg.JobID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create federated registry tokens: %v", err)
	}

	for _, token := range federatedRegistryTokens {
		encHost, bErr := module.BuildTokenEnvVar(token.Hostname)
		if bErr != nil {
			return nil, fmt.Errorf("failed to encode host %s for environment variable: %v",
				token.Hostname, bErr)
		}
		// Set the federated token on the environment since Terraform CLI will need to use
		// it for pulling down any child modules and providers from the federated registry.
		t.fullEnv[encHost] = token.Token
	}

	// Add service account tokens to the host map
	for k, v := range managedIdentitiesResponse.HostCredentialFileMapping {
		// Read the service account token data from the file
		tokenData, rErr := os.ReadFile(v)
		if rErr != nil {
			return nil, fmt.Errorf("failed to read service account token data from file %s: %v", v, rErr)
		}
		encodedHost, rErr := module.BuildTokenEnvVar(k)
		if rErr != nil {
			return nil, fmt.Errorf("failed to encode host %s for environment variable: %v", k, rErr)
		}

		t.fullEnv[encodedHost] = string(tokenData)
	}

	hostTokenMap := map[string]string{}
	for k, v := range t.fullEnv {
		// Add env variables that start with the TF_TOKEN_ prefix to the hostTokenMap
		if strings.HasPrefix(k, tfTokenVarPrefix) {
			hostTokenMap[k] = v
			t.jobLogger.Infof("Setting env variable %s", k)
		}
	}

	// Handle a possible module source (and maybe version).
	var resolvedModuleSource *string
	if t.run.ModuleSource != nil {
		if t.run.ModuleVersion != nil {
			// Registry-style module source; version is always defined in this case
			t.jobLogger.Infof("Resolving module version %s/%s", *t.run.ModuleSource, *t.run.ModuleVersion)

			presignedURL, rErr := resolveModuleSource(*t.run.ModuleSource, *t.run.ModuleVersion, hostTokenMap)
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

	execPath, err := hcInstall.NewInstaller().Ensure(ctx, []src.Source{&fs.ExactVersion{
		Product: product.Terraform,
		Version: t.terraformVersion,
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
		t.deletePathWhenJobCompletes(filepath.Dir(execPath))
	}

	tf, err := tfexec.NewTerraform(t.workspaceDir, execPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tfexec: %v", err)
	}
	// Enable ansi colors
	tf.SetColor(true)

	err = t.setupCredentialHelper(managedIdentitiesResponse.HostCredentialFileMapping)
	if err != nil {
		return nil, err
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

func (t *terraformWorkspace) setupCredentialHelper(hostCredentialFileMapping map[string]string) error {
	isWindows := runtime.GOOS == "windows"
	hasHosts := len(hostCredentialFileMapping) > 0

	if isWindows && hasHosts {
		t.jobLogger.Errorf("Warning: Managed Identity hosts are not supported on windows")
	}

	if isWindows || !hasHosts {
		return nil
	}

	hosts := make([]string, 0, len(hostCredentialFileMapping))
	for host := range hostCredentialFileMapping {
		hosts = append(hosts, host)
	}

	t.jobLogger.Infof("The following managed identity hosts each have a credential file: %v", strings.Join(hosts, ", "))

	credHelperName, err := t.credentialHelper.install(hostCredentialFileMapping)
	if err != nil {
		return err
	}

	err = t.setupCliConfiguration(*credHelperName)
	if err != nil {
		return fmt.Errorf("failed to setup cli configuration: %v", err)
	}

	return nil
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
func (t *terraformWorkspace) setBuiltInEnvVars(_ context.Context, hostCredentialFileMapping map[string]string) error {
	// Set THARSIS_GROUP_PATH
	t.fullEnv["THARSIS_GROUP_PATH"] = t.workspace.FullPath[:strings.LastIndex(t.workspace.FullPath, "/")]

	// Set THARSIS_ENDPOINT which is used by the Terraform Tharsis Provider
	t.fullEnv["THARSIS_ENDPOINT"] = t.jobCfg.APIEndpoint

	err := t.setAPIHostTfTokenEnvVar(hostCredentialFileMapping)
	if err != nil {
		return err
	}

	t.setDiscoveryProtocolHostTfTokenEnvVars(hostCredentialFileMapping)

	return nil
}

func (t *terraformWorkspace) setAPIHostTfTokenEnvVar(hostCredentialFileMapping map[string]string) error {
	apiURL, err := url.Parse(t.jobCfg.APIEndpoint)
	if err != nil {
		return fmt.Errorf("failed to parse API URL %v", err)
	}

	_, hasCredentialFile := hostCredentialFileMapping[apiURL.Host]
	if hasCredentialFile {
		return nil
	}

	apiEncHost, err := module.BuildTokenEnvVar(apiURL.Host)
	if err != nil {
		return fmt.Errorf("failed to encode API URL for environment variable: %v", err)
	}

	t.fullEnv[apiEncHost] = t.jobCfg.JobToken

	return nil
}

func (t *terraformWorkspace) setDiscoveryProtocolHostTfTokenEnvVars(hostCredentialFileMapping map[string]string) {
	for _, host := range t.jobCfg.DiscoveryProtocolHosts {
		_, hasCredentialFile := hostCredentialFileMapping[host]
		if hasCredentialFile {
			continue
		}

		if dpEncHost, err := module.BuildTokenEnvVar(host); err == nil {
			t.fullEnv[dpEncHost] = t.jobCfg.JobToken
		} else {
			t.jobLogger.Errorf("failed to encode Discovery Protocol Host %q: %v", host, err)
		}
	}
}

func (t *terraformWorkspace) deletePathWhenJobCompletes(path string) {
	t.pathsToRemove = append(t.pathsToRemove, path)
}

func (t *terraformWorkspace) createVarsFile() (string, []string, error) {
	// Get all variables in the module
	hclVariables := []types.RunVariable{}
	stringVariables := []types.RunVariable{}
	variablesIncludedInTFConfig := []string{}

	// Parse terraform config
	terraformModule, diagnostics := tfconfig.LoadModule(t.workspaceDir)
	if diagnostics.HasErrors() {
		return "", nil, fmt.Errorf("failed to load terraform module %v", diagnostics)
	}

	// Create map of variables for faster lookup
	inputVarMap := map[string]*types.RunVariable{}
	for _, v := range t.variables {
		inputVarMap[v.Key] = &v
	}

	for _, v := range t.variables {
		if v.Category != types.TerraformVariableCategory {
			continue
		}

		// Check if there is an hcl definition for this variable
		variable, ok := terraformModule.Variables[v.Key]
		if !ok {
			// Make it easier for the user to identity where the variable is coming from.
			if v.NamespacePath != nil && *v.NamespacePath == t.workspace.FullPath {
				t.jobLogger.Warningf("WARNING: Workspace variable %q has a value but is not defined in the terraform module.", v.Key)
			}

			if v.NamespacePath == nil {
				t.jobLogger.Warningf("WARNING: Run variable %q has a value but is not defined in the terraform module.", v.Key)
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

		// Check if wo_version attribute needs to be injected
		variableVersionKey := fmt.Sprintf("%s%s", v.Key, writeOnlyVariableSuffix)
		_, includedInInputVars := inputVarMap[variableVersionKey]
		_, includedInConfig := terraformModule.Variables[variableVersionKey]

		// Only inject the write-only variable value if it's defined in the tf config and not explicitly passed
		// as an input variable
		if !includedInInputVars && includedInConfig {
			if v.VersionID != nil {
				// Since this is a versioned input variable, we need to inject a variable for the hashed version if it's defined
				// in the config but not included in the input variables
				hasher := fnv.New64a()
				hasher.Write([]byte(*v.VersionID))

				// Write-only attribute needs to be an integer so we will use the numerical hashed value
				hashedStr := strconv.FormatUint(hasher.Sum64()%math.MaxInt32, 10)

				hclVariables = append(hclVariables, types.RunVariable{
					Key:      variableVersionKey,
					Value:    &hashedStr,
					Category: types.TerraformVariableCategory,
				})
			} else {
				t.jobLogger.Warningf(
					"WARNING: Variable %s%s will not be injected because write-only version variables are only injected for group/workspace variables and not run scoped variables",
					v.Key,
					writeOnlyVariableSuffix,
				)
			}
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

	filePath := fmt.Sprintf("%s/run-%s.tfvars", t.workspaceDir, t.run.Metadata.ID)
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

func parseTfExecError(err error) *string {
	errStr := err.Error()
	errLines := strings.Split(errStr, "\n")
	// Remove exit status from error message if it exists
	if len(errLines) > 0 && errLines[0] == "exit status 1" {
		errStr = strings.Join(errLines[1:], "\n")
	}
	if errStr != "" {
		return &errStr
	}
	return nil
}
