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

/* Plan Query Resolvers */

// PlanChangesResolver resolves plan changes
type PlanChangesResolver struct {
	planDiff *plan.Diff
}

// Resources resolver
func (r *PlanChangesResolver) Resources() []*plan.ResourceDiff {
	return r.planDiff.Resources
}

// Outputs resolver
func (r *PlanChangesResolver) Outputs() []*plan.OutputDiff {
	return r.planDiff.Outputs
}

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
