package resolver

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/job"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"

	graphql "github.com/graph-gophers/graphql-go"
)

// jobConnectionForRunNode resolves the connection of jobs associated with a run's
// plan or apply node (matched by run ID and job type). Shared by the Plan and
// Apply resolvers. The WorkspaceID gates the query on workspace view permission.
func jobConnectionForRunNode(ctx context.Context, run *models.Run, jobType models.JobType, args *ConnectionQueryArgs) (*JobConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := job.GetJobsInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		WorkspaceID:       &run.WorkspaceID,
		RunID:             &run.Metadata.ID,
		Type:              &jobType,
	}

	if args.Sort != nil {
		sort := db.JobSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewJobConnectionResolver(ctx, &input)
}

// jobConnectionForPolicyCheck resolves the connection of jobs associated with a policy check node
// (its OPA evaluation jobs, including retries), matched by the policy check ID (index-backed by
// index_jobs_on_policy_check_id). The workspaceID gates the query on workspace view permission.
func jobConnectionForPolicyCheck(ctx context.Context, workspaceID, policyCheckID string, args *ConnectionQueryArgs) (*JobConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := job.GetJobsInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		WorkspaceID:       &workspaceID,
		PolicyCheckID:     &policyCheckID,
	}

	if args.Sort != nil {
		sort := db.JobSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewJobConnectionResolver(ctx, &input)
}

/* Plan Query Resolvers */

// PlanChangesResolver resolves plan changes
type PlanChangesResolver struct {
	planDiff *plan.Diff
}

// Resources resolver
func (r *PlanChangesResolver) Resources() []*PlanResourceChangeResolver {
	resolvers := make([]*PlanResourceChangeResolver, len(r.planDiff.Resources))
	for i, d := range r.planDiff.Resources {
		resolvers[i] = &PlanResourceChangeResolver{diff: d}
	}
	return resolvers
}

// Outputs resolver
func (r *PlanChangesResolver) Outputs() []*PlanOutputChangeResolver {
	resolvers := make([]*PlanOutputChangeResolver, len(r.planDiff.Outputs))
	for i, d := range r.planDiff.Outputs {
		resolvers[i] = &PlanOutputChangeResolver{diff: d}
	}
	return resolvers
}

// PlanResourceChangeResolver wraps plan.ResourceDiff
type PlanResourceChangeResolver struct {
	diff *plan.ResourceDiff
}

// Action resolver
func (r *PlanResourceChangeResolver) Action() string { return string(r.diff.Action) }

// Address resolver
func (r *PlanResourceChangeResolver) Address() string { return r.diff.Address }

// Mode resolver
func (r *PlanResourceChangeResolver) Mode() string { return r.diff.Mode }

// ProviderName resolver
func (r *PlanResourceChangeResolver) ProviderName() string { return r.diff.ProviderName }

// ResourceType resolver
func (r *PlanResourceChangeResolver) ResourceType() string { return r.diff.ResourceType }

// ResourceName resolver
func (r *PlanResourceChangeResolver) ResourceName() string { return r.diff.ResourceName }

// ModuleAddress resolver
func (r *PlanResourceChangeResolver) ModuleAddress() string { return r.diff.ModuleAddress }

// UnifiedDiff resolver
func (r *PlanResourceChangeResolver) UnifiedDiff() string { return r.diff.UnifiedDiff }

// OriginalSource resolver
func (r *PlanResourceChangeResolver) OriginalSource() string { return r.diff.OriginalSource }

// Imported resolver
func (r *PlanResourceChangeResolver) Imported() bool { return r.diff.Imported }

// Drifted resolver
func (r *PlanResourceChangeResolver) Drifted() bool { return r.diff.Drifted }

// Moved resolver
func (r *PlanResourceChangeResolver) Moved() bool { return r.diff.Moved }

// Warnings resolver
func (r *PlanResourceChangeResolver) Warnings() []*PlanChangeWarningResolver {
	resolvers := make([]*PlanChangeWarningResolver, len(r.diff.Warnings))
	for i, w := range r.diff.Warnings {
		resolvers[i] = &PlanChangeWarningResolver{warning: w}
	}
	return resolvers
}

// PlanOutputChangeResolver wraps plan.OutputDiff to apply enum casing for GraphQL output.
type PlanOutputChangeResolver struct {
	diff *plan.OutputDiff
}

// Action resolver
func (r *PlanOutputChangeResolver) Action() string { return string(r.diff.Action) }

// OutputName resolver
func (r *PlanOutputChangeResolver) OutputName() string { return r.diff.OutputName }

// UnifiedDiff resolver
func (r *PlanOutputChangeResolver) UnifiedDiff() string { return r.diff.UnifiedDiff }

// OriginalSource resolver
func (r *PlanOutputChangeResolver) OriginalSource() string { return r.diff.OriginalSource }

// Warnings resolver
func (r *PlanOutputChangeResolver) Warnings() []*PlanChangeWarningResolver {
	resolvers := make([]*PlanChangeWarningResolver, len(r.diff.Warnings))
	for i, w := range r.diff.Warnings {
		resolvers[i] = &PlanChangeWarningResolver{warning: w}
	}
	return resolvers
}

// PlanChangeWarningResolver wraps plan.ChangeWarning to apply enum casing for GraphQL output.
type PlanChangeWarningResolver struct {
	warning *plan.ChangeWarning
}

// ChangeType resolver
func (r *PlanChangeWarningResolver) ChangeType() string { return r.warning.ChangeType }

// Line resolver
func (r *PlanChangeWarningResolver) Line() int32 { return r.warning.Line }

// Message resolver
func (r *PlanChangeWarningResolver) Message() string { return r.warning.Message }

// PlanResolver resolves a plan resource
type PlanResolver struct {
	run *models.Run
}

// ID resolver
func (r *PlanResolver) ID() graphql.ID {
	return graphql.ID(r.run.Plan.GetGlobalID())
}

// Status resolver
func (r *PlanResolver) Status() string {
	return string(r.run.Plan.Status)
}

// ErrorMessage resolver
func (r *PlanResolver) ErrorMessage() *string {
	return r.run.Plan.ErrorMessage
}

// HasChanges resolver
func (r *PlanResolver) HasChanges() bool {
	return r.run.Plan.HasChanges
}

// Summary resolver
func (r *PlanResolver) Summary() models.PlanSummary {
	return r.run.Plan.Summary
}

// DiffSize resolver
func (r *PlanResolver) DiffSize() int32 {
	return int32(r.run.Plan.DiffSize)
}

// ResourceAdditions resolver
func (r *PlanResolver) ResourceAdditions() int32 {
	return r.run.Plan.Summary.ResourceAdditions
}

// ResourceChanges resolver
func (r *PlanResolver) ResourceChanges() int32 {
	return r.run.Plan.Summary.ResourceChanges
}

// ResourceDestructions resolver
func (r *PlanResolver) ResourceDestructions() int32 {
	return r.run.Plan.Summary.ResourceDestructions
}

// Metadata resolver
func (r *PlanResolver) Metadata() (*MetadataResolver, error) {
	plan := &r.run.Plan
	return &MetadataResolver{metadata: plan.Metadata(r.run)}, nil
}

// Jobs returns the connection of jobs associated with the plan.
func (r *PlanResolver) Jobs(ctx context.Context, args *ConnectionQueryArgs) (*JobConnectionResolver, error) {
	return jobConnectionForRunNode(ctx, r.run, models.JobPlanType, args)
}

// CurrentJob returns the current job for the plan resource
func (r *PlanResolver) CurrentJob(ctx context.Context) (*JobResolver, error) {
	plan := &r.run.Plan
	if plan.LatestJobID == nil {
		return nil, nil
	}

	job, err := loadJob(ctx, *plan.LatestJobID)
	if err != nil {
		return nil, err
	}

	return &JobResolver{job: job}, nil
}

// Changes resolver
func (r *PlanResolver) Changes(ctx context.Context) (*PlanChangesResolver, error) {
	diff, err := getServiceCatalog(ctx).RunService.GetPlanDiff(ctx, r.run.Plan.ID)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}

	return &PlanChangesResolver{planDiff: diff}, nil
}

// CheckResults resolver
func (r *PlanResolver) CheckResults(ctx context.Context) ([]*TerraformCheckResultResolver, error) {
	results, err := getServiceCatalog(ctx).RunService.GetPlanCheckResults(ctx, r.run.Plan.ID)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return []*TerraformCheckResultResolver{}, nil
		}
		return nil, err
	}

	resolvers := []*TerraformCheckResultResolver{}
	for _, result := range results {
		resultCopy := result
		resolvers = append(resolvers, &TerraformCheckResultResolver{checkResult: &resultCopy})
	}

	return resolvers, nil
}
