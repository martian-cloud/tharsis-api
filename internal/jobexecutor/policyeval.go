package jobexecutor

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/jobclient"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/joblogger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/policyeval"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// maxPolicyBundleEntries caps the number of archive entries processed.
	maxPolicyBundleEntries = 1024
	// maxPolicyBundleBytes caps the total decompressed content across all entries.
	maxPolicyBundleBytes = 50 << 20 // 50 MiB
	// policyInputSchemaVersion is the version of the input document schema exposed to policies.
	// Bump it whenever the shape of the input document changes in a backwards-incompatible way.
	policyInputSchemaVersion = 1
)

// PolicyEvalHandler evaluates the OPA policy sets pinned to a run's policy check, at whichever
// stage (pre-plan, post-plan, pre-apply, post-apply) that check belongs to.
//
// Rego evaluation follows the Conftest convention: rules named "deny" or "violation" (in any
// package) produce violation messages; a non-empty union of those messages across a policy set's
// modules means the policy set FAILED, an empty union means it PASSED. Messages may be strings or
// objects with a "msg" field. Policies are authored in Rego v1 syntax and evaluate against the
// input document assembled in buildInputDocument (schemaVersion, stage, run, tfplan, tfstate).
type PolicyEvalHandler struct {
	client         jobclient.Client
	cancellableCtx context.Context
	run            *pb.Run
	workspace      *pb.Workspace
	job            *pb.Job
	logger         logger.Logger
	jobLogger      joblogger.Logger
}

// evalPolicy is one policy the handler evaluates, taken from a run's policy check. There is one
// entry per policy (the check carries no de-duplication); policyID is echoed back when reporting the
// outcome, and digest — when set — locks the package content. The run snapshots versionConstraint
// unresolved (empty ⇒ latest); resolvedVersion is the concrete version the API resolved it to, and is
// empty until that lookup succeeds.
type evalPolicy struct {
	policyID          string
	policyPath        string
	packageSource     string
	versionConstraint string
	resolvedVersion   string
	digest            string
}

// constraintLabel renders a version constraint for display, naming the empty constraint (which
// resolves to the latest uploaded version) rather than showing nothing.
func constraintLabel(constraint string) string {
	if constraint == "" {
		return "latest"
	}
	return constraint
}

// NewPolicyEvalHandler creates a new PolicyEvalHandler. workspaceDir is accepted to match the
// other job handlers' construction signature; policy evaluation does not use a workspace dir.
func NewPolicyEvalHandler(
	cancellableCtx context.Context,
	workspace *pb.Workspace,
	run *pb.Run,
	job *pb.Job,
	logger logger.Logger,
	jobLogger joblogger.Logger,
	client jobclient.Client,
) (*PolicyEvalHandler, error) {
	return &PolicyEvalHandler{
		cancellableCtx: cancellableCtx,
		workspace:      workspace,
		run:            run,
		job:            job,
		logger:         logger,
		jobLogger:      jobLogger,
		client:         client,
	}, nil
}

// Cleanup is called after the job has been executed. Policy evaluation holds no external
// resources, so there is nothing to clean up.
func (p *PolicyEvalHandler) Cleanup(_ context.Context) error {
	return nil
}

// OnError is called if Execute returns an error. Status is driven by the returned error (the
// executor fails the job, which errors the stage node); this just records a log line.
func (p *PolicyEvalHandler) OnError(_ context.Context, evalErr error) {
	if p.cancellableCtx.Err() != nil {
		p.jobLogger.Errorf("Policy evaluation canceled while in progress %s", failureIcon)
	} else {
		p.jobLogger.Errorf("Error occurred while evaluating policies: %v %s", evalErr, failureIcon)
	}
	p.jobLogger.Flush()
}

// Execute evaluates every policy set attached to the check's stage and reports their outcomes.
func (p *PolicyEvalHandler) Execute(ctx context.Context) error {
	opaData := p.job.GetOpaData()
	if opaData == nil {
		return fmt.Errorf("job %s is missing OPA job data", p.job.Metadata.Id)
	}
	policyCheckID := opaData.PolicyCheckId

	check := p.opaCheck(policyCheckID)
	if check == nil {
		return fmt.Errorf("run %s has no OPA policy check node to report policy outcomes for", p.run.Metadata.Id)
	}

	// Policy evaluation only reads variable values/keys for OPA input; it never needs actual
	// sensitive values.
	runVariables, err := p.client.GetRunVariables(ctx, p.run.Metadata.Id, false)
	if err != nil {
		return fmt.Errorf("failed to get run variables: %w", err)
	}

	policies, err := policiesFromCheck(check)
	if err != nil {
		return fmt.Errorf("failed to parse policies %w", err)
	}
	if len(policies) == 0 {
		p.jobLogger.Infof("No policy sets are attached to this run; nothing to evaluate.")
		if err = p.client.ReportRunPolicyOutcomes(ctx, policyCheckID, nil); err != nil {
			return fmt.Errorf("failed to report empty stage outcomes: %w", err)
		}
		return nil
	}

	// A pre-plan check runs before the plan, so there is no plan JSON to evaluate — the input
	// carries only the run/workspace context. Every other stage (post-plan, pre-apply, post-apply)
	// additionally includes the tfplan.
	var planJSON interface{}
	if check.StageName != pb.RunTaskStageName_RUN_TASK_STAGE_NAME_PRE_PLAN {
		planJSON, err = p.downloadPlanJSON(ctx)
		if err != nil {
			return err
		}
	}

	// A post-apply check evaluates after state has been written, so it additionally carries the
	// state's JSON representation, taken from the state version this run created.
	var stateJSON interface{}
	if check.StageName == pb.RunTaskStageName_RUN_TASK_STAGE_NAME_POST_APPLY {
		stateJSON, err = p.downloadStateJSON(ctx)
		if err != nil {
			return err
		}
	}

	input := p.buildInputDocument(check.StageName, runVariables, planJSON, stateJSON)

	// Resolve and download every policy set package up front. A constraint no uploaded version
	// satisfies, a package whose pinned digest does not match its content, and a package whose files
	// are missing are each recorded as a failed outcome (skipping evaluation) so they flow through the
	// server's enforcement-level verdict. bundles[i] == nil means "do not evaluate".
	outcomes := make([]jobclient.RunPolicyOutcomeInput, len(policies))
	bundles := make([]*policyeval.Bundle, len(policies))
	for i := range policies {
		// Taken by pointer so the version the constraint resolves to is recorded on the policy and
		// available to every message below.
		ps := &policies[i]
		outcomes[i] = jobclient.RunPolicyOutcomeInput{PolicyID: ps.policyID}

		// Resolve the policy's version constraint to a concrete uploaded version and its package
		// version id, then download it by id. The run snapshots the constraint rather than a version,
		// so this is where a policy tracking a range picks up a newly published version. A constraint
		// no uploaded version satisfies — including a deleted package — surfaces as NotFound, which we
		// record as a failed outcome so it flows through the server's enforcement-level verdict.
		pv, dErr := p.client.GetPackageVersion(ctx, ps.packageSource, ps.versionConstraint)
		if dErr != nil {
			if status.Code(dErr) == codes.NotFound {
				msg := fmt.Sprintf("Package source %q with version constraint %q not found",
					ps.packageSource, constraintLabel(ps.versionConstraint))
				outcomes[i].Passed = false
				outcomes[i].Messages = []string{msg}
				continue
			}
			return fmt.Errorf("failed to resolve policy set %s: %w", ps.policyPath, dErr)
		}
		ps.resolvedVersion = pv.Version

		// The resolved version is not recorded on the run, so this log line is the record of which
		// policy code the check actually evaluated.
		p.jobLogger.Infof("Package %q: version constraint %s resolved to version %s with digest %s",
			ps.packageSource, constraintLabel(ps.versionConstraint), ps.resolvedVersion, pv.ShaSum)

		rc, dErr := p.client.DownloadPackage(ctx, pv.Metadata.Id)
		if dErr != nil {
			if status.Code(dErr) == codes.NotFound {
				msg := fmt.Sprintf("policy %s has missing files: the package version has been deleted", ps.policyPath)
				p.jobLogger.Errorf("%s %s", msg, failureIcon)
				outcomes[i].Passed = false
				outcomes[i].Messages = []string{msg}
				continue
			}
			return fmt.Errorf("failed to download package for policy %s: %w", ps.policyPath, dErr)
		}

		bundle, eErr := extractPolicyBundle(rc)
		if eErr != nil {
			return fmt.Errorf("failed to extract policy package %s: %w", ps.policyPath, eErr)
		}

		// Verify integrity after extraction: extractPolicyBundle computes the digest of the raw
		// package stream it read, so the pinned digest is checked against the exact bytes that were
		// unpacked. Extraction is already bounded (entry/byte caps), so unpacking before verifying
		// cannot exhaust memory.
		if ps.digest != "" && bundle.ShaSum != ps.digest {
			msg := fmt.Sprintf("Policy %s package content digest mismatch: expected %s, got %s",
				ps.policyPath, ps.digest, bundle.ShaSum)
			p.jobLogger.Errorf("%s %s", msg, failureIcon)
			outcomes[i].Passed = false
			outcomes[i].Messages = []string{msg}
			continue
		}

		bundles[i] = &bundle
	}

	// Evaluate each policy set that passed the digest gate in parallel. Each goroutine builds its own
	// rego object (rego objects are not safe for concurrent use).
	eg, egCtx := errgroup.WithContext(ctx)
	for i := range policies {
		if bundles[i] == nil {
			continue
		}
		i := i

		bundle := *bundles[i]
		eg.Go(func() error {
			violations, evalErr := policyeval.Evaluate(egCtx, bundle, input)
			if evalErr != nil {
				outcomes[i].Messages = []string{evalErr.Error()}
				outcomes[i].Passed = false
				return nil
			}

			passed := len(violations) == 0
			if !passed {
				// Violations are collected by walking the evaluated data document, so their order
				// across rego packages is however Go happened to range a map. Sorted here because
				// each one is reported as its own message and the UI renders them in order — an
				// unchanged policy should not shuffle its findings between attempts.
				slices.Sort(violations)
				outcomes[i].Messages = violations
			}

			outcomes[i].Passed = passed
			return nil
		})
	}

	if err = eg.Wait(); err != nil {
		return err
	}

	for i, ps := range policies {
		if outcomes[i].Passed {
			p.jobLogger.Infof("Policy %s passed %s", ps.policyPath, successIcon)
		} else {
			// One violation per line in the log, where the newline is only formatting — the
			// messages are reported to the API as a list.
			p.jobLogger.Errorf("Policy %s failed %s\n%s", ps.policyPath, failureIcon, strings.Join(outcomes[i].Messages, "\n"))
		}
	}

	p.jobLogger.Flush()

	// Report once with all outcomes. The server computes the stage verdict from the run's pinned
	// enforcement levels; the executor does not set the node status here.
	if err = p.client.ReportRunPolicyOutcomes(ctx, policyCheckID, outcomes); err != nil {
		return fmt.Errorf("failed to report stage outcomes: %w", err)
	}

	return nil
}

// downloadPlanJSON downloads and unmarshals the run's plan JSON (the Terraform plan document).
func (p *PolicyEvalHandler) downloadPlanJSON(ctx context.Context) (interface{}, error) {
	var buf bytes.Buffer
	if err := p.client.DownloadPlanJSON(ctx, p.run.PlanId, &buf); err != nil {
		return nil, fmt.Errorf("failed to download plan JSON: %w", err)
	}

	if buf.Len() == 0 {
		return nil, nil
	}

	var planJSON interface{}
	if err := json.Unmarshal(buf.Bytes(), &planJSON); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan JSON: %w", err)
	}

	return planJSON, nil
}

// downloadStateJSON downloads and unmarshals the JSON representation ("terraform show -json") of the
// state version this run's apply created. It is keyed off the run, not off the workspace's current
// state version: those diverge as soon as anything else writes state after the apply (a rollback or a
// direct state push), and the policy must be evaluated against what this run actually produced.
//
// It returns nil, without an error, whenever no representation is available: the run wrote no state at
// all, or its state version has no JSON rendering stored because it was written by a job executor
// predating that upload, or by a client that never produces one. The key is then omitted from the input
// document rather than failing the check, so a mixed-version fleet keeps evaluating policies while
// runners roll out. The trade-off is real and deliberate: a rule that reads input.tfstate is simply
// undefined when the representation is missing, so it neither fires nor errors. The absence is logged
// to the job log so it is diagnosable from the run.
func (p *PolicyEvalHandler) downloadStateJSON(ctx context.Context) (interface{}, error) {
	stateVersion, err := p.client.GetRunStateVersion(ctx, p.run.Metadata.Id)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			p.jobLogger.Infof("This run created no state version; policies will be evaluated without input.tfstate.")
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get the run's state version: %w", err)
	}

	stateVersionID := stateVersion.GetMetadata().GetId()

	var buf bytes.Buffer
	if err := p.client.DownloadStateVersionJSON(ctx, stateVersionID, &buf); err != nil {
		if errors.Is(err, client.ErrStateVersionJSONNotFound) {
			p.jobLogger.Infof(
				"State version %s has no JSON representation stored; policies will be evaluated without input.tfstate.",
				stateVersionID,
			)
			return nil, nil
		}
		return nil, fmt.Errorf("failed to download state version JSON: %w", err)
	}

	if buf.Len() == 0 {
		p.jobLogger.Infof(
			"State version %s has an empty JSON representation; policies will be evaluated without input.tfstate.",
			stateVersionID,
		)
		return nil, nil
	}

	var stateJSON interface{}
	if err := json.Unmarshal(buf.Bytes(), &stateJSON); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state JSON: %w", err)
	}

	return stateJSON, nil
}

// buildInputDocument assembles the rich input document evaluated by the policies. Keys are left
// absent when the corresponding value is not applicable — notably tfplan, which is absent only for a
// pre-plan stage (which runs before the plan exists), and tfstate, which is present only for a
// post-apply stage (the only stage that evaluates after state has been written) and only when a JSON
// representation of that state is available. The stage is surfaced at the top level so a policy can
// branch on which stage it is evaluating at.
//
// Both tfplan and tfstate are the documented "terraform show -json" renderings, not Terraform's
// internal file formats, so a policy reads tfplan.resource_changes and
// tfstate.values.root_module.resources.
func (p *PolicyEvalHandler) buildInputDocument(
	stage pb.RunTaskStageName,
	runVariables []*pb.RunVariable,
	planJSON interface{},
	stateJSON interface{},
) map[string]interface{} {
	runDoc := map[string]interface{}{
		"id":                     p.run.Metadata.Id,
		"createdBy":              p.run.CreatedBy,
		"isDestroy":              p.run.IsDestroy,
		"refresh":                p.run.Refresh,
		"refreshOnly":            p.run.RefreshOnly,
		"speculative":            p.run.Speculative,
		"isAssessmentRun":        p.run.IsAssessmentRun,
		"terraformVersion":       p.run.TerraformVersion,
		"targetAddresses":        p.run.TargetAddresses,
		"moduleSource":           p.run.ModuleSource,
		"moduleVersion":          p.run.ModuleVersion,
		"moduleDigest":           p.run.ModuleDigest,
		"configurationVersionId": p.run.ConfigurationVersionId,
	}
	if p.workspace != nil {
		runDoc["workspacePath"] = p.workspace.FullPath
		runDoc["workspaceId"] = p.workspace.Metadata.Id
	}

	variables := make([]map[string]interface{}, len(runVariables))
	for i, v := range runVariables {
		entry := map[string]interface{}{
			"key":       v.Key,
			"category":  v.Category,
			"sensitive": v.Sensitive,
		}
		if v.NamespacePath != nil {
			entry["namespacePath"] = *v.NamespacePath
		}
		if v.Value != nil {
			entry["value"] = *v.Value
		}
		variables[i] = entry
	}

	runDoc["variables"] = variables

	input := map[string]interface{}{
		"schemaVersion": policyInputSchemaVersion,
		"stage":         strings.ToLower(strings.TrimPrefix(stage.String(), "RUN_TASK_STAGE_NAME_")),
		"run":           runDoc,
	}

	if planJSON != nil {
		input["tfplan"] = planJSON
	}
	if stateJSON != nil {
		input["tfstate"] = stateJSON
	}

	return input
}

// opaCheck returns the run's OPA policy check node with the given ID (at any stage), or nil
// if the run has no such node. The run carries its checks nested under task stages, so this walks
// every stage's checks.
func (p *PolicyEvalHandler) opaCheck(policyCheckID string) *pb.PolicyCheck {
	for _, stage := range p.run.TaskStages {
		for _, check := range stage.PolicyChecks {
			if check.Id == policyCheckID {
				return check
			}
		}
	}
	return nil
}

// policiesFromCheck maps a policy check's policies into the set the handler evaluates — one per
// policy, keyed by policy id.
func policiesFromCheck(check *pb.PolicyCheck) ([]evalPolicy, error) {
	policies := make([]evalPolicy, 0, len(check.Policies))
	for _, policy := range check.Policies {
		policyTRN, err := trn.ParseAny(policy.Provenance.PolicyTrn)
		if err != nil {
			return nil, err
		}
		policies = append(policies, evalPolicy{
			policyID:          policy.Id,
			policyPath:        policyTRN.Path(),
			packageSource:     policy.PackageSource,
			versionConstraint: policy.PackageVersionConstraint,
			digest:            policy.GetPackageDigest(),
		})
	}
	return policies, nil
}

// extractPolicyBundle reads a policy set package (tar.gz) into its rego modules and JSON data
// documents. The reader is closed before returning.
//
// The bundle is untrusted input from the package registry, so extraction is bounded: at most
// maxPolicyBundleEntries archive entries and maxPolicyBundleBytes of total decompressed content.
// Without these caps a small but highly compressible archive (a decompression bomb) could exhaust
// the policy-eval job container's memory. Exceeding either bound aborts extraction with an error.
func extractPolicyBundle(rc io.ReadCloser) (policyeval.Bundle, error) {
	defer rc.Close()

	bundle := policyeval.Bundle{
		Modules: map[string]string{},
		Data:    map[string]interface{}{},
	}

	// Hash the raw (compressed) package bytes as they are consumed, so the returned bundle carries
	// the digest of the exact stream it was extracted from. gzip reads through the tee, and the
	// stream is fully drained after extraction so the digest covers every byte.
	hasher := sha256.New()
	tee := io.TeeReader(rc, hasher)

	gz, err := gzip.NewReader(tee)
	if err != nil {
		return bundle, fmt.Errorf("failed to open package gzip stream: %w", err)
	}
	defer gz.Close()

	// Budget for total decompressed bytes across all files, shared so many small files can't
	// collectively blow past the cap the way a per-file limit alone would allow.
	remaining := int64(maxPolicyBundleBytes)
	var tr *tar.Reader
	// readEntry reads the current tar entry, failing closed if it would push total decompressed
	// output past the budget. It reads one byte past the remaining budget so an exact-fit file is
	// still accepted while an over-budget one is detected.
	readEntry := func() ([]byte, error) {
		contents, err := io.ReadAll(io.LimitReader(tr, remaining+1))
		if err != nil {
			return nil, err
		}
		if int64(len(contents)) > remaining {
			return nil, fmt.Errorf(
				"policy package exceeds maximum decompressed size of %d bytes", maxPolicyBundleBytes)
		}
		remaining -= int64(len(contents))
		return contents, nil
	}

	tr = tar.NewReader(gz)
	entries := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return bundle, fmt.Errorf("failed to read package tar stream: %w", err)
		}

		entries++
		if entries > maxPolicyBundleEntries {
			return bundle, fmt.Errorf(
				"policy package exceeds maximum of %d entries", maxPolicyBundleEntries)
		}

		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		name := hdr.Name
		switch {
		case strings.HasSuffix(name, ".rego"):
			contents, err := readEntry()
			if err != nil {
				return bundle, fmt.Errorf("failed to read rego file %s: %w", name, err)
			}
			bundle.Modules[name] = string(contents)
		case strings.HasSuffix(name, ".json"):
			contents, err := readEntry()
			if err != nil {
				return bundle, fmt.Errorf("failed to read data file %s: %w", name, err)
			}
			var doc map[string]interface{}
			if err := json.Unmarshal(contents, &doc); err != nil {
				// Not a JSON object of data values; skip it.
				continue
			}
			for k, v := range doc {
				bundle.Data[k] = v
			}
		}
	}

	if len(bundle.Modules) == 0 {
		return bundle, fmt.Errorf("policy package contains no .rego modules")
	}

	// Drain any bytes gzip did not consume (e.g. trailing padding after the tar EOF) so the hash
	// covers the entire raw stream, matching a digest taken over the full downloaded package.
	if _, err := io.Copy(io.Discard, tee); err != nil {
		return bundle, fmt.Errorf("failed to read package stream: %w", err)
	}
	bundle.ShaSum = hex.EncodeToString(hasher.Sum(nil))

	return bundle, nil
}
