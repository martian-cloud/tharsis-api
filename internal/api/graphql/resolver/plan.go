package resolver

import (
	"context"
	"strconv"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"

	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
)

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
	plan *models.Plan
}

// ID resolver
func (r *PlanResolver) ID() graphql.ID {
	return graphql.ID(gid.ToGlobalID(gid.PlanType, r.plan.Metadata.ID))
}

// Status resolver
func (r *PlanResolver) Status() string {
	return string(r.plan.Status)
}

// ErrorMessage resolver
func (r *PlanResolver) ErrorMessage() *string {
	return r.plan.ErrorMessage
}

// HasChanges resolver
func (r *PlanResolver) HasChanges() bool {
	return bool(r.plan.HasChanges)
}

// Summary resolver
func (r *PlanResolver) Summary() models.PlanSummary {
	return r.plan.Summary
}

// DiffSize resolver
func (r *PlanResolver) DiffSize() int32 {
	return int32(r.plan.PlanDiffSize)
}

// ResourceAdditions resolver
func (r *PlanResolver) ResourceAdditions() int32 {
	return r.plan.Summary.ResourceAdditions
}

// ResourceChanges resolver
func (r *PlanResolver) ResourceChanges() int32 {
	return r.plan.Summary.ResourceChanges
}

// ResourceDestructions resolver
func (r *PlanResolver) ResourceDestructions() int32 {
	return r.plan.Summary.ResourceDestructions
}

// Metadata resolver
func (r *PlanResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.plan.Metadata}
}

// CurrentJob returns the current job for the plan resource
func (r *PlanResolver) CurrentJob(ctx context.Context) (*JobResolver, error) {
	job, err := getRunService(ctx).GetLatestJobForPlan(ctx, r.plan.Metadata.ID)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}
	return &JobResolver{job: job}, nil
}

// Changes resolver
func (r *PlanResolver) Changes(ctx context.Context) (*PlanChangesResolver, error) {
	diff, err := getRunService(ctx).GetPlanDiff(ctx, r.plan.Metadata.ID)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}

	return &PlanChangesResolver{planDiff: diff}, nil
}

/* Plan Mutation Resolvers */

// PlanMutationPayload is the response payload for plan mutation
type PlanMutationPayload struct {
	ClientMutationID *string
	Plan             *models.Plan
	Problems         []Problem
}

// PlanMutationPayloadResolver resolves a PlanMutationPayload
type PlanMutationPayloadResolver struct {
	PlanMutationPayload
}

// Plan field resolver
func (r *PlanMutationPayloadResolver) Plan() *PlanResolver {
	if r.PlanMutationPayload.Plan == nil {
		return nil
	}
	return &PlanResolver{plan: r.PlanMutationPayload.Plan}
}

// UpdatePlanInput contains the input for updating a plan
type UpdatePlanInput struct {
	ClientMutationID *string
	ID               string
	Metadata         *MetadataInput
	Status           string
	HasChanges       bool
	ErrorMessage     *string
}

func handlePlanMutationProblem(e error, clientMutationID *string) (*PlanMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := PlanMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &PlanMutationPayloadResolver{PlanMutationPayload: payload}, nil
}

func updatePlanMutation(ctx context.Context, input *UpdatePlanInput) (*PlanMutationPayloadResolver, error) {
	runService := getRunService(ctx)

	plan, err := runService.GetPlan(ctx, gid.FromGlobalID(input.ID))
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, cErr := strconv.Atoi(input.Metadata.Version)
		if cErr != nil {
			return nil, cErr
		}

		plan.Metadata.Version = v
	}

	// Update fields
	plan.Status = models.PlanStatus(input.Status)
	plan.HasChanges = input.HasChanges
	plan.ErrorMessage = input.ErrorMessage

	plan, err = runService.UpdatePlan(ctx, plan)
	if err != nil {
		return nil, err
	}

	payload := PlanMutationPayload{ClientMutationID: input.ClientMutationID, Plan: plan, Problems: []Problem{}}
	return &PlanMutationPayloadResolver{PlanMutationPayload: payload}, nil
}

/* Plan loader */

const planLoaderKey = "plan"

// RegisterPlanLoader registers a plan loader function
func RegisterPlanLoader(collection *loader.Collection) {
	collection.Register(planLoaderKey, planBatchFunc)
}

func loadPlan(ctx context.Context, id string) (*models.Plan, error) {
	ldr, err := loader.Extract(ctx, planLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	ws, ok := data.(models.Plan)
	if !ok {
		return nil, errors.New("Wrong type")
	}

	return &ws, nil
}

func planBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	service := getRunService(ctx)

	plans, err := service.GetPlansByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range plans {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}
