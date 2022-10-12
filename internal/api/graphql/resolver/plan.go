package resolver

import (
	"context"
	"strconv"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"

	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
)

/* Plan Query Resolvers */

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

// HasChanges resolver
func (r *PlanResolver) HasChanges() bool {
	return bool(r.plan.HasChanges)
}

// Metadata resolver
func (r *PlanResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.plan.Metadata}
}

// ResourceAdditions resolver
func (r *PlanResolver) ResourceAdditions() int32 {
	return int32(r.plan.ResourceAdditions)
}

// ResourceChanges resolver
func (r *PlanResolver) ResourceChanges() int32 {
	return int32(r.plan.ResourceChanges)
}

// ResourceDestructions resolver
func (r *PlanResolver) ResourceDestructions() int32 {
	return int32(r.plan.ResourceDestructions)
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
func (r *PlanMutationPayloadResolver) Plan(ctx context.Context) *PlanResolver {
	if r.PlanMutationPayload.Plan == nil {
		return nil
	}
	return &PlanResolver{plan: r.PlanMutationPayload.Plan}
}

// UpdatePlanInput contains the input for updating a plan
type UpdatePlanInput struct {
	ClientMutationID     *string
	ID                   string
	Metadata             *MetadataInput
	Status               string
	HasChanges           bool
	ResourceAdditions    int32
	ResourceChanges      int32
	ResourceDestructions int32
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
	plan.ResourceAdditions = int(input.ResourceAdditions)
	plan.ResourceChanges = int(input.ResourceChanges)
	plan.ResourceDestructions = int(input.ResourceDestructions)

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
		return nil, errors.NewError(errors.EInternal, "Wrong type")
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
