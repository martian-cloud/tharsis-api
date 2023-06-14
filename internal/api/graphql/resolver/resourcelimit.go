package resolver

import (
	"context"
	"strconv"

	graphql "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/resourcelimit"
)

/* ResourceLimits Query Resolvers */

// ResourceLimitResolver resolves a resource limit
type ResourceLimitResolver struct {
	resourceLimit *models.ResourceLimit
}

func resourceLimitsQuery(ctx context.Context) ([]*ResourceLimitResolver, error) {

	resourceLimits, err := getResourceLimitService(ctx).GetResourceLimits(ctx)
	if err != nil {
		return nil, err
	}

	results := []*ResourceLimitResolver{}
	for _, limit := range resourceLimits {
		copyLimit := limit
		results = append(results, &ResourceLimitResolver{
			resourceLimit: &copyLimit,
		})
	}

	return results, nil
}

// ID resolver
func (r *ResourceLimitResolver) ID() graphql.ID {
	return graphql.ID(gid.ToGlobalID(gid.ResourceLimitType, r.resourceLimit.Metadata.ID))
}

// Metadata resolver
func (r *ResourceLimitResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.resourceLimit.Metadata}
}

// Name resolver
func (r *ResourceLimitResolver) Name() string {
	return r.resourceLimit.Name
}

// Value resolver
func (r *ResourceLimitResolver) Value() int32 {
	return int32(r.resourceLimit.Value)
}

/* Resource Limit Mutation Resolvers */

// ResourceLimitMutationPayload is the response payload for a resource limit mutation
type ResourceLimitMutationPayload struct {
	ClientMutationID *string
	ResourceLimit    *models.ResourceLimit
	Problems         []Problem
}

// ResourceLimitMutationPayloadResolver resolves a ResourceLimitMutationPayload
type ResourceLimitMutationPayloadResolver struct {
	ResourceLimitMutationPayload
}

// ResourceLimit field resolver
func (r *ResourceLimitMutationPayloadResolver) ResourceLimit() *ResourceLimitResolver {
	if r.ResourceLimitMutationPayload.ResourceLimit == nil {
		return nil
	}
	return &ResourceLimitResolver{resourceLimit: r.ResourceLimitMutationPayload.ResourceLimit}
}

// UpdateResourceLimitInput contains the input for updating a resource limit
type UpdateResourceLimitInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	Name             string
	Value            int32
}

func handleResourceLimitMutationProblem(e error, clientMutationID *string) (*ResourceLimitMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := ResourceLimitMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &ResourceLimitMutationPayloadResolver{ResourceLimitMutationPayload: payload}, nil
}

func updateResourceLimitMutation(ctx context.Context, input *UpdateResourceLimitInput) (*ResourceLimitMutationPayloadResolver, error) {
	resourceLimitService := getResourceLimitService(ctx)

	toUpdate := &resourcelimit.UpdateResourceLimitInput{
		Name:  input.Name,
		Value: int(input.Value),
	}
	if input.Metadata != nil {
		v, err := strconv.Atoi(input.Metadata.Version)
		toUpdate.MetadataVersion = &v
		if err != nil {
			return nil, err
		}
	}

	resourceLimit, err := resourceLimitService.UpdateResourceLimit(ctx, toUpdate)
	if err != nil {
		return nil, err
	}

	payload := ResourceLimitMutationPayload{ClientMutationID: input.ClientMutationID, ResourceLimit: resourceLimit, Problems: []Problem{}}
	return &ResourceLimitMutationPayloadResolver{ResourceLimitMutationPayload: payload}, nil
}
