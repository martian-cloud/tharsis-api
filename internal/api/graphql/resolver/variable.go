package resolver

import (
	"context"

	"github.com/aws/smithy-go/ptr"
	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/variable"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// NamesapceVariableVersionQueryArgs are used to query a variable version
type NamesapceVariableVersionQueryArgs struct {
	ID                    string
	IncludeSensitiveValue *bool
}

// NamespaceVariableVersionConnectionQueryArgs are used to query a version connection
type NamespaceVariableVersionConnectionQueryArgs struct {
	ConnectionQueryArgs
}

// NamespaceVariableVersionQueryArgs are used to query a single version
type NamespaceVariableVersionQueryArgs struct {
	Name string
}

// NamespaceVariableVersionEdgeResolver resolves version edges
type NamespaceVariableVersionEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *NamespaceVariableVersionEdgeResolver) Cursor() (string, error) {
	version, ok := r.edge.Node.(models.VariableVersion)
	if !ok {
		return "", errors.New("Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&version)
	return *cursor, err
}

// Node returns a version node
func (r *NamespaceVariableVersionEdgeResolver) Node() (*NamespaceVariableVersionResolver, error) {
	version, ok := r.edge.Node.(models.VariableVersion)
	if !ok {
		return nil, errors.New("Failed to convert node type")
	}

	return &NamespaceVariableVersionResolver{version: &version}, nil
}

// NamespaceVariableVersionConnectionResolver resolves a version connection
type NamespaceVariableVersionConnectionResolver struct {
	connection Connection
}

// NewNamespaceVariableVersionConnectionResolver creates a new NamespaceVariableVersionConnectionResolver
func NewNamespaceVariableVersionConnectionResolver(ctx context.Context, input *variable.GetVariableVersionsInput) (*NamespaceVariableVersionConnectionResolver, error) {
	versionService := getVariableService(ctx)

	result, err := versionService.GetVariableVersions(ctx, input)
	if err != nil {
		return nil, err
	}

	versions := result.VariableVersions

	// Create edges
	edges := make([]Edge, len(versions))
	for i, version := range versions {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: version}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(versions) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&versions[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&versions[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &NamespaceVariableVersionConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *NamespaceVariableVersionConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *NamespaceVariableVersionConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *NamespaceVariableVersionConnectionResolver) Edges() *[]*NamespaceVariableVersionEdgeResolver {
	resolvers := make([]*NamespaceVariableVersionEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &NamespaceVariableVersionEdgeResolver{edge: edge}
	}
	return &resolvers
}

// NamespaceVariableVersionResolver resolves a variable version
type NamespaceVariableVersionResolver struct {
	version *models.VariableVersion
}

// ID resolver
func (r *NamespaceVariableVersionResolver) ID() graphql.ID {
	return graphql.ID(gid.ToGlobalID(gid.VariableVersionType, r.version.Metadata.ID))
}

// Metadata resolver
func (r *NamespaceVariableVersionResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.version.Metadata}
}

// Hcl resolver
func (r *NamespaceVariableVersionResolver) Hcl() *bool {
	return &r.version.Hcl
}

// Key resolver
func (r *NamespaceVariableVersionResolver) Key() string {
	return r.version.Key
}

// Value resolver
func (r *NamespaceVariableVersionResolver) Value() *string {
	return r.version.Value
}

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

// Sensitive resolver
func (r *NamespaceVariableResolver) Sensitive() bool {
	return r.variable.Sensitive
}

// Hcl resolver
// DEPRECATED: Hcl is deprecated and will be removed in a future release
func (r *NamespaceVariableResolver) Hcl() *bool {
	return &r.variable.Hcl
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

// LatestVersionID resolver
func (r *NamespaceVariableResolver) LatestVersionID() string {
	return gid.ToGlobalID(gid.VariableVersionType, r.variable.LatestVersionID)
}

// Metadata resolver
func (r *NamespaceVariableResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.variable.Metadata}
}

// Versions resolver
func (r *NamespaceVariableResolver) Versions(ctx context.Context, args *NamespaceVariableVersionConnectionQueryArgs) (*NamespaceVariableVersionConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := variable.GetVariableVersionsInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		VariableID:        r.variable.Metadata.ID,
	}

	if args.Sort != nil {
		sort := db.VariableVersionSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewNamespaceVariableVersionConnectionResolver(ctx, &input)
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

func namespaceVariableVersionQuery(ctx context.Context, args *NamesapceVariableVersionQueryArgs) (*NamespaceVariableVersionResolver, error) {
	variableService := getVariableService(ctx)

	includeSensitiveValue := false
	if args.IncludeSensitiveValue != nil {
		includeSensitiveValue = *args.IncludeSensitiveValue
	}

	version, err := variableService.GetVariableVersionByID(ctx, gid.FromGlobalID(args.ID), includeSensitiveValue)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}

	return &NamespaceVariableVersionResolver{version: version}, nil
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
	Sensitive        *bool
	Key              string
	Value            string
	// DEPRECATED: to be removed in a future release.
	Hcl *bool
}

// UpdateNamespaceVariableInput is the input for updating a variable
type UpdateNamespaceVariableInput struct {
	ClientMutationID *string
	ID               string
	Key              string
	Value            string
	// DEPRECATED: to be removed in a future release.
	Hcl *bool
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
		Sensitive *bool
		Key       string
		Value     string
		// DEPRECATED: to be removed in a future release.
		Hcl *bool
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
	variables := []*variable.SetVariablesInputVariable{}

	for _, v := range input.Variables {
		varableInput := &variable.SetVariablesInputVariable{
			Hcl:   ptr.ToBool(v.Hcl),
			Key:   v.Key,
			Value: v.Value,
		}

		if v.Sensitive != nil {
			varableInput.Sensitive = *v.Sensitive
		}

		variables = append(variables, varableInput)
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
	options := &variable.CreateVariableInput{
		NamespacePath: input.NamespacePath,
		Category:      models.VariableCategory(input.Category),
		Hcl:           ptr.ToBool(input.Hcl),
		Key:           input.Key,
		Value:         input.Value,
	}

	if input.Sensitive != nil {
		options.Sensitive = *input.Sensitive
	}

	variable, err := getVariableService(ctx).CreateVariable(ctx, options)
	if err != nil {
		return nil, err
	}

	payload := VariableMutationPayload{ClientMutationID: input.ClientMutationID, NamespacePath: &variable.NamespacePath, Problems: []Problem{}}
	return &VariableMutationPayloadResolver{VariableMutationPayload: payload}, nil
}

func updateNamespaceVariableMutation(ctx context.Context, input *UpdateNamespaceVariableInput) (*VariableMutationPayloadResolver, error) {
	service := getVariableService(ctx)

	updatedVar, err := service.UpdateVariable(ctx, &variable.UpdateVariableInput{
		ID:    gid.FromGlobalID(input.ID),
		Key:   input.Key,
		Value: input.Value,
		Hcl:   ptr.ToBool(input.Hcl),
	})
	if err != nil {
		return nil, err
	}

	payload := VariableMutationPayload{ClientMutationID: input.ClientMutationID, NamespacePath: &updatedVar.NamespacePath, Problems: []Problem{}}
	return &VariableMutationPayloadResolver{VariableMutationPayload: payload}, nil
}

func deleteNamespaceVariableMutation(ctx context.Context, input *DeleteNamespaceVariableInput) (*VariableMutationPayloadResolver, error) {
	service := getVariableService(ctx)

	variableModel, err := service.GetVariableByID(ctx, gid.FromGlobalID(input.ID))
	if err != nil {
		return nil, err
	}

	if err := service.DeleteVariable(ctx, &variable.DeleteVariableInput{
		ID: variableModel.Metadata.ID,
	}); err != nil {
		return nil, err
	}

	payload := VariableMutationPayload{ClientMutationID: input.ClientMutationID, NamespacePath: &variableModel.NamespacePath, Problems: []Problem{}}
	return &VariableMutationPayloadResolver{VariableMutationPayload: payload}, nil
}

/* NamespaceVariable loader */

const namespaceVariableLoaderKey = "namespaceVariable"

// RegisterNamespaceVariableLoader registers a namespaceVariable loader function
func RegisterNamespaceVariableLoader(collection *loader.Collection) {
	collection.Register(namespaceVariableLoaderKey, namespaceVariableBatchFunc)
}

func loadVariable(ctx context.Context, id string) (*models.Variable, error) {
	ldr, err := loader.Extract(ctx, namespaceVariableLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	namespaceVariable, ok := data.(models.Variable)
	if !ok {
		return nil, errors.New("Wrong type")
	}

	return &namespaceVariable, nil
}

func namespaceVariableBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	variables, err := getVariableService(ctx).GetVariablesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range variables {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}
