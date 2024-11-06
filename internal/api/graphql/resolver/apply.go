package resolver

import (
	"context"
	"strconv"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"

	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
)

/* Apply Query Resolvers */

// ApplyResolver resolves a apply resource
type ApplyResolver struct {
	apply *models.Apply
}

// ID resolver
func (r *ApplyResolver) ID() graphql.ID {
	return graphql.ID(gid.ToGlobalID(gid.ApplyType, r.apply.Metadata.ID))
}

// Status resolver
func (r *ApplyResolver) Status() string {
	return string(r.apply.Status)
}

// ErrorMessage resolver
func (r *ApplyResolver) ErrorMessage() *string {
	return r.apply.ErrorMessage
}

// TriggeredBy resolver
func (r *ApplyResolver) TriggeredBy() *string {
	if r.apply.TriggeredBy == "" {
		return nil
	}
	return &r.apply.TriggeredBy
}

// Metadata resolver
func (r *ApplyResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.apply.Metadata}
}

// CurrentJob returns the current job for the apply resource
func (r *ApplyResolver) CurrentJob(ctx context.Context) (*JobResolver, error) {
	job, err := getRunService(ctx).GetLatestJobForApply(ctx, r.apply.Metadata.ID)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}
	return &JobResolver{job: job}, nil
}

// Comment resolver
func (r *ApplyResolver) Comment() string {
	return r.apply.Comment
}

/* Apply Mutation Resolvers */

// ApplyMutationPayload is the response payload for an apply mutation
type ApplyMutationPayload struct {
	ClientMutationID *string
	Apply            *models.Apply
	Problems         []Problem
}

// ApplyMutationPayloadResolver resolves a ApplyMutationPayload
type ApplyMutationPayloadResolver struct {
	ApplyMutationPayload
}

// Apply field resolver
func (r *ApplyMutationPayloadResolver) Apply() *ApplyResolver {
	if r.ApplyMutationPayload.Apply == nil {
		return nil
	}
	return &ApplyResolver{apply: r.ApplyMutationPayload.Apply}
}

// UpdateApplyInput contains the input for updating an apply
type UpdateApplyInput struct {
	ClientMutationID *string
	ID               string
	Metadata         *MetadataInput
	Status           string
	ErrorMessage     *string
}

func handleApplyMutationProblem(e error, clientMutationID *string) (*ApplyMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := ApplyMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &ApplyMutationPayloadResolver{ApplyMutationPayload: payload}, nil
}

func updateApplyMutation(ctx context.Context, input *UpdateApplyInput) (*ApplyMutationPayloadResolver, error) {
	runService := getRunService(ctx)

	apply, err := runService.GetApply(ctx, gid.FromGlobalID(input.ID))
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, cErr := strconv.Atoi(input.Metadata.Version)
		if cErr != nil {
			return nil, cErr
		}

		apply.Metadata.Version = v
	}

	// Update fields
	apply.Status = models.ApplyStatus(input.Status)
	apply.ErrorMessage = input.ErrorMessage

	apply, err = runService.UpdateApply(ctx, apply)
	if err != nil {
		return nil, err
	}

	payload := ApplyMutationPayload{ClientMutationID: input.ClientMutationID, Apply: apply, Problems: []Problem{}}
	return &ApplyMutationPayloadResolver{ApplyMutationPayload: payload}, nil
}

/* Apply loader */

const applyLoaderKey = "apply"

// RegisterApplyLoader registers an apply loader function
func RegisterApplyLoader(collection *loader.Collection) {
	collection.Register(applyLoaderKey, applyBatchFunc)
}

func loadApply(ctx context.Context, id string) (*models.Apply, error) {
	ldr, err := loader.Extract(ctx, applyLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	ws, ok := data.(models.Apply)
	if !ok {
		return nil, errors.New("Wrong type")
	}

	return &ws, nil
}

func applyBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	service := getRunService(ctx)

	applies, err := service.GetAppliesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range applies {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}
