package resolver

import (
	"context"

	graphql "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/variable"
)

// NamespaceVariableResolver resolves a variable resource
type NamespaceVariableResolver struct {
	variable *models.Variable
}

// ID resolver
func (r *NamespaceVariableResolver) ID() graphql.ID {
	return graphql.ID(gid.ToGlobalID(gid.VariableType, r.variable.Metadata.ID))
}

// Category resolver
func (r *NamespaceVariableResolver) Category() string {
	return string(r.variable.Category)
}

// Hcl resolver
func (r *NamespaceVariableResolver) Hcl() bool {
	return r.variable.Hcl
}

// NamespacePath resolver
func (r *NamespaceVariableResolver) NamespacePath() string {
	return r.variable.NamespacePath
}

// Key resolver
func (r *NamespaceVariableResolver) Key() string {
	return r.variable.Key
}

// Value resolver
func (r *NamespaceVariableResolver) Value() *string {
	return r.variable.Value
}

// Metadata resolver
func (r *NamespaceVariableResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.variable.Metadata}
}

/* Variable Queries */

func getVariables(ctx context.Context, namespacePath string) ([]*NamespaceVariableResolver, error) {
	service := getVariableService(ctx)

	result, err := service.GetVariables(ctx, namespacePath)
	if err != nil {
		return nil, err
	}

	resolvers := []*NamespaceVariableResolver{}
	for _, v := range result {
		varCopy := v
		resolvers = append(resolvers, &NamespaceVariableResolver{variable: &varCopy})
	}

	return resolvers, nil
}

/* Variable Mutations */

// VariableMutationPayload is the response payload for a variable mutation
type VariableMutationPayload struct {
	ClientMutationID *string
	NamespacePath    *string
	Problems         []Problem
}

// VariableMutationPayloadResolver resolves a VariableMutationPayload
type VariableMutationPayloadResolver struct {
	VariableMutationPayload
}

// Namespace field resolver
func (r *VariableMutationPayloadResolver) Namespace(ctx context.Context) (*NamespaceResolver, error) {
	if r.VariableMutationPayload.NamespacePath == nil {
		return nil, nil
	}
	group, err := getGroupService(ctx).GetGroupByFullPath(ctx, *r.NamespacePath)
	if err != nil && errors.ErrorCode(err) != errors.ENotFound {
		return nil, err
	}
	if group != nil {
		return &NamespaceResolver{result: &GroupResolver{group: group}}, nil
	}

	ws, err := getWorkspaceService(ctx).GetWorkspaceByFullPath(ctx, *r.NamespacePath)
	if err != nil {
		return nil, err
	}
	return &NamespaceResolver{result: &WorkspaceResolver{workspace: ws}}, nil
}

// CreateNamespaceVariableInput is the input for creating a variable
type CreateNamespaceVariableInput struct {
	ClientMutationID *string
	NamespacePath    string
	Category         string
	Key              string
	Value            string
	Hcl              bool
}

// UpdateNamespaceVariableInput is the input for updating a variable
type UpdateNamespaceVariableInput struct {
	ClientMutationID *string
	ID               string
	Key              string
	Value            string
	Hcl              bool
}

// DeleteNamespaceVariableInput is the input for deleting a variable
type DeleteNamespaceVariableInput struct {
	ClientMutationID *string
	ID               string
}

// SetNamespaceVariablesInput is the input for setting namespace variables
type SetNamespaceVariablesInput struct {
	ClientMutationID *string
	NamespacePath    string
	Category         models.VariableCategory
	Variables        []struct {
		Key   string
		Value string
		Hcl   bool
	}
}

func handleVariableMutationProblem(e error, clientMutationID *string) (*VariableMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := VariableMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &VariableMutationPayloadResolver{VariableMutationPayload: payload}, nil
}

func setNamespaceVariablesMutation(ctx context.Context, input *SetNamespaceVariablesInput) (*VariableMutationPayloadResolver, error) {
	variables := []models.Variable{}

	for _, v := range input.Variables {
		vCopy := v
		variables = append(variables, models.Variable{
			Hcl:           v.Hcl,
			Key:           v.Key,
			Value:         &vCopy.Value,
			Category:      input.Category,
			NamespacePath: input.NamespacePath,
		})
	}

	if err := getVariableService(ctx).SetVariables(ctx, &variable.SetVariablesInput{
		NamespacePath: input.NamespacePath,
		Category:      input.Category,
		Variables:     variables,
	}); err != nil {
		return nil, err
	}

	payload := VariableMutationPayload{ClientMutationID: input.ClientMutationID, NamespacePath: &input.NamespacePath, Problems: []Problem{}}
	return &VariableMutationPayloadResolver{VariableMutationPayload: payload}, nil
}

func createNamespaceVariableMutation(ctx context.Context, input *CreateNamespaceVariableInput) (*VariableMutationPayloadResolver, error) {
	variable, err := getVariableService(ctx).CreateVariable(ctx, &models.Variable{
		NamespacePath: input.NamespacePath,
		Category:      models.VariableCategory(input.Category),
		Hcl:           input.Hcl,
		Key:           input.Key,
		Value:         &input.Value,
	})
	if err != nil {
		return nil, err
	}

	payload := VariableMutationPayload{ClientMutationID: input.ClientMutationID, NamespacePath: &variable.NamespacePath, Problems: []Problem{}}
	return &VariableMutationPayloadResolver{VariableMutationPayload: payload}, nil
}

func updateNamespaceVariableMutation(ctx context.Context, input *UpdateNamespaceVariableInput) (*VariableMutationPayloadResolver, error) {
	service := getVariableService(ctx)

	variable, err := service.GetVariableByID(ctx, gid.FromGlobalID(input.ID))
	if err != nil {
		return nil, err
	}

	variable.Hcl = input.Hcl
	variable.Key = input.Key
	variable.Value = &input.Value

	updatedVar, err := service.UpdateVariable(ctx, variable)
	if err != nil {
		return nil, err
	}

	payload := VariableMutationPayload{ClientMutationID: input.ClientMutationID, NamespacePath: &updatedVar.NamespacePath, Problems: []Problem{}}
	return &VariableMutationPayloadResolver{VariableMutationPayload: payload}, nil
}

func deleteNamespaceVariableMutation(ctx context.Context, input *DeleteNamespaceVariableInput) (*VariableMutationPayloadResolver, error) {
	service := getVariableService(ctx)

	variable, err := service.GetVariableByID(ctx, gid.FromGlobalID(input.ID))
	if err != nil {
		return nil, err
	}

	if err := service.DeleteVariable(ctx, variable); err != nil {
		return nil, err
	}

	payload := VariableMutationPayload{ClientMutationID: input.ClientMutationID, NamespacePath: &variable.NamespacePath, Problems: []Problem{}}
	return &VariableMutationPayloadResolver{VariableMutationPayload: payload}, nil
}
