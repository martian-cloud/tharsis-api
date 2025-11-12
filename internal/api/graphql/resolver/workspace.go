package resolver

import (
	"context"
	"strconv"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/serviceaccount"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/vcs"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"

	"github.com/aws/smithy-go/ptr"
	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
)

/* Workspace Query Resolvers */

// WorkspaceConnectionQueryArgs are used to query a workspace connection
type WorkspaceConnectionQueryArgs struct {
	ConnectionQueryArgs
	GroupPath   *string // DEPRECATED: use GroupID instead with a TRN
	GroupID     *string
	Search      *string
	LabelFilter *WorkspaceLabelsFilter
}

// WorkspaceLabelsFilter represents label filtering criteria
type WorkspaceLabelsFilter struct {
	Labels []WorkspaceLabelInput
}

// WorkspaceLabelResolver resolves a workspace label resource (JSONB-based)
type WorkspaceLabelResolver struct {
	key   string // For JSONB-based labels
	value string // For JSONB-based labels
}

// Key resolver
func (r *WorkspaceLabelResolver) Key() string {
	return r.key
}

// Value resolver
func (r *WorkspaceLabelResolver) Value() string {
	return r.value
}

// WorkspaceLabelInput represents a label key-value pair
type WorkspaceLabelInput struct {
	Key   string
	Value string
}

// WorkspaceQueryArgs are used to query a single workspace
// DEPRECATED: use node query instead with a TRN
type WorkspaceQueryArgs struct {
	FullPath string
}

// WorkspaceEdgeResolver resolves workspace edges
type WorkspaceEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *WorkspaceEdgeResolver) Cursor() (string, error) {
	workspace, ok := r.edge.Node.(models.Workspace)
	if !ok {
		return "", errors.New("Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&workspace)
	return *cursor, err
}

// Node returns a workspace node
func (r *WorkspaceEdgeResolver) Node() (*WorkspaceResolver, error) {
	workspace, ok := r.edge.Node.(models.Workspace)
	if !ok {
		return nil, errors.New("Failed to convert node type")
	}

	return &WorkspaceResolver{workspace: &workspace}, nil
}

// WorkspaceConnectionResolver resolves a workspace connection
type WorkspaceConnectionResolver struct {
	connection Connection
}

// NewWorkspaceConnectionResolver creates a new WorkspaceConnectionResolver
func NewWorkspaceConnectionResolver(ctx context.Context, input *workspace.GetWorkspacesInput) (*WorkspaceConnectionResolver, error) {
	workspaceService := getServiceCatalog(ctx).WorkspaceService

	result, err := workspaceService.GetWorkspaces(ctx, input)
	if err != nil {
		return nil, err
	}

	workspaces := result.Workspaces

	// Create edges
	edges := make([]Edge, len(workspaces))
	for i, workspace := range workspaces {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: workspace}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(workspaces) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&workspaces[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&workspaces[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &WorkspaceConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *WorkspaceConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *WorkspaceConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *WorkspaceConnectionResolver) Edges() *[]*WorkspaceEdgeResolver {
	resolvers := make([]*WorkspaceEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &WorkspaceEdgeResolver{edge: edge}
	}
	return &resolvers
}

// WorkspaceResolver resolves a workspace resource
type WorkspaceResolver struct {
	workspace *models.Workspace
}

// ID resolver
func (r *WorkspaceResolver) ID() graphql.ID {
	return graphql.ID(r.workspace.GetGlobalID())
}

// Name resolver
func (r *WorkspaceResolver) Name() string {
	return r.workspace.Name
}

// GroupPath resolver
func (r *WorkspaceResolver) GroupPath() string {
	return r.workspace.GetGroupPath()
}

// FullPath resolver
func (r *WorkspaceResolver) FullPath() string {
	return r.workspace.FullPath
}

// Description resolver
func (r *WorkspaceResolver) Description() string {
	return r.workspace.Description
}

// Metadata resolver
func (r *WorkspaceResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.workspace.Metadata}
}

// Group resolver
func (r *WorkspaceResolver) Group(ctx context.Context) (*GroupResolver, error) {
	group, err := loadGroup(ctx, r.workspace.GroupID)
	if err != nil {
		return nil, err
	}

	return &GroupResolver{group: group}, nil
}

// Memberships resolver
// The field is called "memberships", but most everything else is called "namespace memberships".
func (r *WorkspaceResolver) Memberships(ctx context.Context) ([]*NamespaceMembershipResolver, error) {
	resolvers := []*NamespaceMembershipResolver{}

	result, err := getServiceCatalog(ctx).NamespaceMembershipService.GetNamespaceMembershipsForNamespace(ctx, r.workspace.FullPath)
	if err != nil {
		return nil, err
	}

	for _, v := range result {
		varCopy := v
		resolvers = append(resolvers, &NamespaceMembershipResolver{namespaceMembership: &varCopy})
	}

	return resolvers, nil
}

// Variables resolver
func (r *WorkspaceResolver) Variables(ctx context.Context) ([]*NamespaceVariableResolver, error) {
	return getVariables(ctx, r.workspace.FullPath)
}

// AssignedManagedIdentities resolver
func (r *WorkspaceResolver) AssignedManagedIdentities(ctx context.Context) ([]*ManagedIdentityResolver, error) {
	identities, err := getServiceCatalog(ctx).ManagedIdentityService.GetManagedIdentitiesForWorkspace(ctx, r.workspace.Metadata.ID)
	if err != nil {
		return nil, err
	}

	resolvers := []*ManagedIdentityResolver{}
	for _, identity := range identities {
		identityCopy := identity
		resolvers = append(resolvers, &ManagedIdentityResolver{managedIdentity: &identityCopy})
	}

	return resolvers, nil
}

// Labels resolver - converts JSONB labels to GraphQL label array
func (r *WorkspaceResolver) Labels(ctx context.Context) ([]*WorkspaceLabelResolver, error) {
	if len(r.workspace.Labels) == 0 {
		return []*WorkspaceLabelResolver{}, nil
	}

	resolvers := make([]*WorkspaceLabelResolver, 0, len(r.workspace.Labels))
	for key, value := range r.workspace.Labels {
		// Create a simple label structure for JSONB approach
		label := &WorkspaceLabelResolver{
			key:   key,
			value: value,
		}
		resolvers = append(resolvers, label)
	}

	return resolvers, nil
}

// CurrentJob resolver
func (r *WorkspaceResolver) CurrentJob(ctx context.Context) (*JobResolver, error) {
	if r.workspace.CurrentJobID == "" {
		return nil, nil
	}

	job, err := loadJob(ctx, r.workspace.CurrentJobID)
	if err != nil {
		return nil, err
	}

	return &JobResolver{job: job}, nil
}

// DirtyState resolver
func (r *WorkspaceResolver) DirtyState() bool {
	return r.workspace.DirtyState
}

// Locked resolver
func (r *WorkspaceResolver) Locked() bool {
	return r.workspace.Locked
}

// ServiceAccounts resolver
func (r *WorkspaceResolver) ServiceAccounts(ctx context.Context, args *ServiceAccountsConnectionQueryArgs) (*ServiceAccountConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := &serviceaccount.GetServiceAccountsInput{
		PaginationOptions: &pagination.Options{
			First:  args.First,
			Last:   args.Last,
			Before: args.Before,
			After:  args.After,
		},
		Search:        args.Search,
		NamespacePath: r.workspace.FullPath,
	}

	if args.IncludeInherited != nil && *args.IncludeInherited {
		input.IncludeInherited = true
	}

	return NewServiceAccountConnectionResolver(ctx, input)
}

// ManagedIdentities resolver
func (r *WorkspaceResolver) ManagedIdentities(ctx context.Context, args *ManagedIdentityConnectionQueryArgs) (*ManagedIdentityConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := managedidentity.GetManagedIdentitiesInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		Search:            args.Search,
		NamespacePath:     r.workspace.FullPath,
	}

	if args.Sort != nil {
		sort := db.ManagedIdentitySortableField(*args.Sort)
		input.Sort = &sort
	}

	if args.IncludeInherited != nil && *args.IncludeInherited {
		input.IncludeInherited = true
	}

	return NewManagedIdentityConnectionResolver(ctx, &input)
}

// CurrentStateVersion resolver
func (r *WorkspaceResolver) CurrentStateVersion(ctx context.Context) (*StateVersionResolver, error) {
	// Current state version can be empty string, so return a nil resolver.
	if r.workspace.CurrentStateVersionID == "" {
		return nil, nil
	}

	currentStateVersion, err := loadStateVersion(ctx, r.workspace.CurrentStateVersionID)
	if err != nil {
		return nil, err
	}

	return &StateVersionResolver{stateVersion: currentStateVersion}, nil
}

// StateVersions resolver
func (r *WorkspaceResolver) StateVersions(ctx context.Context, args *StateVersionConnectionQueryArgs) (*StateVersionConnectionResolver, error) {
	sort := db.StateVersionSortableFieldUpdatedAtDesc
	input := &workspace.GetStateVersionsInput{
		Sort: &sort,
		PaginationOptions: &pagination.Options{
			First:  args.First,
			Last:   args.Last,
			Before: args.Before,
			After:  args.After,
		},
		Workspace: r.workspace,
	}

	return NewStateVersionConnectionResolver(ctx, input)
}

// MaxJobDuration resolver
func (r *WorkspaceResolver) MaxJobDuration() int32 {
	return *r.workspace.MaxJobDuration
}

// CreatedBy resolver
func (r *WorkspaceResolver) CreatedBy() string {
	return r.workspace.CreatedBy
}

// TerraformVersion resolver
func (r *WorkspaceResolver) TerraformVersion() string {
	return r.workspace.TerraformVersion
}

// Assessment resolver
func (r *WorkspaceResolver) Assessment(ctx context.Context) (*WorkspaceAssessmentResolver, error) {
	assessment, err := loadWorkspaceAssessment(ctx, r.workspace.Metadata.ID)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}
	return &WorkspaceAssessmentResolver{assessment: assessment}, nil
}

// ActivityEvents resolver
func (r *WorkspaceResolver) ActivityEvents(ctx context.Context,
	args *ActivityEventConnectionQueryArgs,
) (*ActivityEventConnectionResolver, error) {
	input, err := getActivityEventsInputFromQueryArgs(ctx, args)
	if err != nil {
		// error is already a Tharsis error
		return nil, err
	}

	// Need to filter to this workspace/namespace.
	input.NamespacePath = &r.workspace.FullPath

	return NewActivityEventConnectionResolver(ctx, input)
}

// VCSProviders resolver
func (r *WorkspaceResolver) VCSProviders(ctx context.Context, args *VCSProviderConnectionQueryArgs) (*VCSProviderConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := vcs.GetVCSProvidersInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		Search:            args.Search,
		NamespacePath:     r.workspace.FullPath,
	}

	if args.Sort != nil {
		sort := db.VCSProviderSortableField(*args.Sort)
		input.Sort = &sort
	}

	if args.IncludeInherited != nil && *args.IncludeInherited {
		input.IncludeInherited = true
	}

	return NewVCSProviderConnectionResolver(ctx, &input)
}

// WorkspaceVCSProviderLink resolver
func (r *WorkspaceResolver) WorkspaceVCSProviderLink(ctx context.Context) (*WorkspaceVCSProviderLinkResolver, error) {
	link, err := getServiceCatalog(ctx).VCSService.GetWorkspaceVCSProviderLinkByWorkspaceID(ctx, r.workspace.Metadata.ID)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}

	return &WorkspaceVCSProviderLinkResolver{workspaceVCSProviderLink: link}, nil
}

// PreventDestroyPlan resolver
func (r *WorkspaceResolver) PreventDestroyPlan() bool {
	return r.workspace.PreventDestroyPlan
}

// VCSEvents resolver
func (r *WorkspaceResolver) VCSEvents(ctx context.Context, args *VCSEventConnectionQueryArgs) (*VCSEventConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := vcs.GetVCSEventsInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		WorkspaceID:       r.workspace.Metadata.ID,
	}

	if args.Sort != nil {
		sort := db.VCSEventSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewVCSEventConnectionResolver(ctx, &input)
}

// RunnerTags resolver
func (r *WorkspaceResolver) RunnerTags(ctx context.Context) (*namespace.RunnerTagsSetting, error) {
	return getServiceCatalog(ctx).WorkspaceService.GetRunnerTagsSetting(ctx, r.workspace)
}

// DriftDetectionEnabled resolver
func (r *WorkspaceResolver) DriftDetectionEnabled(ctx context.Context) (*namespace.DriftDetectionEnabledSetting, error) {
	return getServiceCatalog(ctx).WorkspaceService.GetDriftDetectionEnabledSetting(ctx, r.workspace)
}

// DEPRECATED: use node query instead
func workspaceQuery(ctx context.Context, args *WorkspaceQueryArgs) (*WorkspaceResolver, error) {
	ws, err := getServiceCatalog(ctx).WorkspaceService.GetWorkspaceByTRN(ctx, types.WorkspaceModelType.BuildTRN(args.FullPath))
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}

	return &WorkspaceResolver{workspace: ws}, nil
}

func workspacesQuery(ctx context.Context, args *WorkspaceConnectionQueryArgs) (*WorkspaceConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := workspace.GetWorkspacesInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		Search:            args.Search,
	}

	if args.GroupID != nil && args.GroupPath != nil {
		return nil, errors.New("either groupId or groupPath must be specified, not both", errors.WithErrorCode(errors.EInvalid))
	}

	var groupValueToResolve *string
	if args.GroupID != nil {
		groupValueToResolve = args.GroupID
	} else if args.GroupPath != nil {
		groupValueToResolve = ptr.String(types.GroupModelType.BuildTRN(*args.GroupPath))
	}

	if groupValueToResolve != nil {
		groupID, err := getServiceCatalog(ctx).FetchModelID(ctx, *groupValueToResolve)
		if err != nil {
			return nil, err
		}

		input.GroupID = &groupID
	}

	// Handle label filtering
	if args.LabelFilter != nil && len(args.LabelFilter.Labels) > 0 {
		labelFilters := make([]db.WorkspaceLabelFilter, len(args.LabelFilter.Labels))
		for i, label := range args.LabelFilter.Labels {
			labelFilters[i] = db.WorkspaceLabelFilter{
				Key:   label.Key,
				Value: label.Value,
			}
		}
		input.LabelFilters = labelFilters
	}

	if args.Sort != nil {
		sort := db.WorkspaceSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewWorkspaceConnectionResolver(ctx, &input)
}

/* Workspace Mutation Resolvers */

// WorkspaceMutationPayload is the response payload for a workspace mutation
type WorkspaceMutationPayload struct {
	ClientMutationID *string
	Workspace        *models.Workspace
	Problems         []Problem
}

// WorkspaceMutationPayloadResolver resolves a WorkspaceMutationPayload
type WorkspaceMutationPayloadResolver struct {
	WorkspaceMutationPayload
}

// Workspace field resolver
func (r *WorkspaceMutationPayloadResolver) Workspace() *WorkspaceResolver {
	if r.WorkspaceMutationPayload.Workspace == nil {
		return nil
	}
	return &WorkspaceResolver{workspace: r.WorkspaceMutationPayload.Workspace}
}

// CreateWorkspaceInput contains the input for creating a new workspace
type CreateWorkspaceInput struct {
	ClientMutationID      *string
	MaxJobDuration        *int32
	TerraformVersion      *string
	PreventDestroyPlan    *bool
	RunnerTags            *NamespaceRunnerTagsInput
	Name                  string
	GroupID               *string
	GroupPath             *string // DEPRECATED: use GroupID instead with a TRN
	Description           string
	DriftDetectionEnabled *NamespaceDriftDetectionEnabledInput
	Labels                *[]WorkspaceLabelInput
}

// UpdateWorkspaceInput contains the input for updating a workspace
// Find the workspace via either ID or WorkspacePath.
// Modify the other fields.
type UpdateWorkspaceInput struct {
	ClientMutationID      *string
	Metadata              *MetadataInput
	MaxJobDuration        *int32
	TerraformVersion      *string
	Description           *string
	PreventDestroyPlan    *bool
	WorkspacePath         *string // DEPRECATED: use ID instead with a TRN
	ID                    *string
	RunnerTags            *NamespaceRunnerTagsInput
	DriftDetectionEnabled *NamespaceDriftDetectionEnabledInput
	Labels                *[]WorkspaceLabelInput
}

// DeleteWorkspaceInput contains the input for deleting a workspace
type DeleteWorkspaceInput struct {
	ClientMutationID *string
	Force            *bool
	Metadata         *MetadataInput
	WorkspacePath    *string // DEPRECATED: use ID instead with a TRN
	ID               *string
}

// LockWorkspaceInput contains the input for locking a workspace
type LockWorkspaceInput struct {
	ClientMutationID *string
	WorkspacePath    *string // DEPRECATED: use WorkspaceID instead with a TRN
	WorkspaceID      *string
}

// UnlockWorkspaceInput contains the input for unlocking a workspace
type UnlockWorkspaceInput struct {
	ClientMutationID *string
	WorkspacePath    *string // DEPRECATED: use WorkspaceID instead with a TRN
	WorkspaceID      *string
}

// DestroyWorkspaceInput contains the input for destroying a workspace
type DestroyWorkspaceInput struct {
	ClientMutationID *string
	WorkspacePath    *string // DEPRECATED: use WorkspaceID instead with a TRN
	WorkspaceID      *string
}

// AssessWorkspaceInput contains the input for running a workspace assessment
type AssessWorkspaceInput struct {
	ClientMutationID *string
	WorkspacePath    *string // DEPRECATED: use WorkspaceID instead with a TRN
	WorkspaceID      *string
}

// MigrateWorkspaceInput contains the input for migrating a workspace
type MigrateWorkspaceInput struct {
	ClientMutationID *string
	NewGroupPath     *string // DEPRECATED: use NewGroupID instead with a TRN
	WorkspacePath    *string // DEPRECATED: use WorkspaceID instead with a TRN
	WorkspaceID      *string
	NewGroupID       *string
}

func handleWorkspaceMutationProblem(e error, clientMutationID *string) (*WorkspaceMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := WorkspaceMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &WorkspaceMutationPayloadResolver{WorkspaceMutationPayload: payload}, nil
}

func createWorkspaceMutation(ctx context.Context, input *CreateWorkspaceInput) (*WorkspaceMutationPayloadResolver, error) {
	groupID, err := toModelID(ctx, input.GroupPath, input.GroupID, types.GroupModelType)
	if err != nil {
		return nil, err
	}

	var terraformVersion string
	if input.TerraformVersion != nil {
		terraformVersion = *input.TerraformVersion
	}

	// Default PreventDestroyPlan to false if not specified.
	preventDestroyPlan := false
	if input.PreventDestroyPlan != nil {
		preventDestroyPlan = *input.PreventDestroyPlan
	}

	wsCreateOptions := models.Workspace{
		Name:               input.Name,
		GroupID:            groupID,
		Description:        input.Description,
		MaxJobDuration:     input.MaxJobDuration,
		TerraformVersion:   terraformVersion,
		PreventDestroyPlan: preventDestroyPlan,
	}

	// Handle labels input
	if input.Labels != nil && len(*input.Labels) > 0 {
		labels := make(map[string]string, len(*input.Labels))
		for _, label := range *input.Labels {
			labels[label.Key] = label.Value
		}
		wsCreateOptions.Labels = labels
	}

	if input.RunnerTags != nil {
		if err = input.RunnerTags.Validate(); err != nil {
			return nil, err
		}

		if input.RunnerTags.Tags != nil {
			wsCreateOptions.RunnerTags = *input.RunnerTags.Tags
		}
	}

	if input.DriftDetectionEnabled != nil {
		if err = input.DriftDetectionEnabled.Validate(); err != nil {
			return nil, err
		}

		if input.DriftDetectionEnabled.Enabled != nil {
			wsCreateOptions.EnableDriftDetection = input.DriftDetectionEnabled.Enabled
		}
	}

	createdWorkspace, err := getServiceCatalog(ctx).WorkspaceService.CreateWorkspace(ctx, &wsCreateOptions)
	if err != nil {
		return nil, err
	}

	payload := WorkspaceMutationPayload{ClientMutationID: input.ClientMutationID, Workspace: createdWorkspace, Problems: []Problem{}}
	return &WorkspaceMutationPayloadResolver{WorkspaceMutationPayload: payload}, nil
}

func updateWorkspaceMutation(ctx context.Context, input *UpdateWorkspaceInput) (*WorkspaceMutationPayloadResolver, error) {
	workspaceID, err := toModelID(ctx, input.WorkspacePath, input.ID, types.WorkspaceModelType)
	if err != nil {
		return nil, err
	}

	wsService := getServiceCatalog(ctx).WorkspaceService

	ws, err := wsService.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, cErr := strconv.Atoi(input.Metadata.Version)
		if cErr != nil {
			return nil, cErr
		}

		ws.Metadata.Version = v
	}

	if d := input.MaxJobDuration; d != nil {
		ws.MaxJobDuration = d
	}

	// Update Terraform Version if specified.
	if input.TerraformVersion != nil {
		ws.TerraformVersion = *input.TerraformVersion
	}

	// Update fields
	if input.Description != nil {
		ws.Description = *input.Description
	}

	// Update PreventDestroyPlan if specified.
	if input.PreventDestroyPlan != nil {
		ws.PreventDestroyPlan = *input.PreventDestroyPlan
	}

	if input.RunnerTags != nil {
		if tErr := input.RunnerTags.Validate(); tErr != nil {
			return nil, tErr
		}

		if input.RunnerTags.Tags != nil {
			ws.RunnerTags = *input.RunnerTags.Tags
		}

		if input.RunnerTags.Inherit {
			ws.RunnerTags = nil
		}
	}

	if input.DriftDetectionEnabled != nil {
		if err = input.DriftDetectionEnabled.Validate(); err != nil {
			return nil, err
		}

		if input.DriftDetectionEnabled.Enabled != nil {
			ws.EnableDriftDetection = input.DriftDetectionEnabled.Enabled
		}

		if input.DriftDetectionEnabled.Inherit {
			ws.EnableDriftDetection = nil
		}
	}

	// Handle labels input
	if input.Labels != nil {
		labels := make(map[string]string, len(*input.Labels))
		for _, label := range *input.Labels {
			labels[label.Key] = label.Value
		}
		ws.Labels = labels
	}

	ws, err = wsService.UpdateWorkspace(ctx, ws)
	if err != nil {
		return nil, err
	}

	payload := WorkspaceMutationPayload{ClientMutationID: input.ClientMutationID, Workspace: ws, Problems: []Problem{}}
	return &WorkspaceMutationPayloadResolver{WorkspaceMutationPayload: payload}, nil
}

func deleteWorkspaceMutation(ctx context.Context, input *DeleteWorkspaceInput) (*WorkspaceMutationPayloadResolver, error) {
	workspaceID, err := toModelID(ctx, input.WorkspacePath, input.ID, types.WorkspaceModelType)
	if err != nil {
		return nil, err
	}

	wsService := getServiceCatalog(ctx).WorkspaceService

	ws, err := wsService.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, err := strconv.Atoi(input.Metadata.Version)
		if err != nil {
			return nil, err
		}

		ws.Metadata.Version = v
	}

	force := false
	if input.Force != nil {
		force = *input.Force
	}

	if err := wsService.DeleteWorkspace(ctx, ws, force); err != nil {
		return nil, err
	}

	payload := WorkspaceMutationPayload{ClientMutationID: input.ClientMutationID, Workspace: ws, Problems: []Problem{}}
	return &WorkspaceMutationPayloadResolver{WorkspaceMutationPayload: payload}, nil
}

func lockWorkspaceMutation(ctx context.Context, input *LockWorkspaceInput) (*WorkspaceMutationPayloadResolver, error) {
	workspaceID, err := toModelID(ctx, input.WorkspacePath, input.WorkspaceID, types.WorkspaceModelType)
	if err != nil {
		return nil, err
	}

	wsService := getServiceCatalog(ctx).WorkspaceService

	ws, err := wsService.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	ws, err = wsService.LockWorkspace(ctx, ws)
	if err != nil {
		return nil, err
	}

	payload := WorkspaceMutationPayload{ClientMutationID: input.ClientMutationID, Workspace: ws, Problems: []Problem{}}
	return &WorkspaceMutationPayloadResolver{WorkspaceMutationPayload: payload}, nil
}

func unlockWorkspaceMutation(ctx context.Context, input *UnlockWorkspaceInput) (*WorkspaceMutationPayloadResolver, error) {
	workspaceID, err := toModelID(ctx, input.WorkspacePath, input.WorkspaceID, types.WorkspaceModelType)
	if err != nil {
		return nil, err
	}

	wsService := getServiceCatalog(ctx).WorkspaceService

	ws, err := wsService.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	ws, err = wsService.UnlockWorkspace(ctx, ws)
	if err != nil {
		return nil, err
	}

	payload := WorkspaceMutationPayload{ClientMutationID: input.ClientMutationID, Workspace: ws, Problems: []Problem{}}
	return &WorkspaceMutationPayloadResolver{WorkspaceMutationPayload: payload}, nil
}

func destroyWorkspaceMutation(ctx context.Context, input *DestroyWorkspaceInput) (*RunMutationPayloadResolver, error) {
	workspaceID, err := toModelID(ctx, input.WorkspacePath, input.WorkspaceID, types.WorkspaceModelType)
	if err != nil {
		return nil, err
	}

	run, err := getServiceCatalog(ctx).RunService.CreateDestroyRunForWorkspace(ctx, &run.CreateDestroyRunForWorkspaceInput{
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, err
	}

	payload := RunMutationPayload{ClientMutationID: input.ClientMutationID, Run: run, Problems: []Problem{}}
	return &RunMutationPayloadResolver{RunMutationPayload: payload}, nil
}

func assessWorkspaceMutation(ctx context.Context, input *AssessWorkspaceInput) (*RunMutationPayloadResolver, error) {
	workspaceID, err := toModelID(ctx, input.WorkspacePath, input.WorkspaceID, types.WorkspaceModelType)
	if err != nil {
		return nil, err
	}

	run, err := getServiceCatalog(ctx).RunService.CreateAssessmentRunForWorkspace(ctx, &run.CreateAssessmentRunForWorkspaceInput{
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, err
	}

	payload := RunMutationPayload{ClientMutationID: input.ClientMutationID, Run: run, Problems: []Problem{}}
	return &RunMutationPayloadResolver{RunMutationPayload: payload}, nil
}

func migrateWorkspaceMutation(ctx context.Context, input *MigrateWorkspaceInput) (*WorkspaceMutationPayloadResolver, error) {
	workspaceID, err := toModelID(ctx, input.WorkspacePath, input.WorkspaceID, types.WorkspaceModelType)
	if err != nil {
		return nil, err
	}

	groupID, err := toModelID(ctx, input.NewGroupPath, input.NewGroupID, types.GroupModelType)
	if err != nil {
		return nil, err
	}

	workspace, err := getServiceCatalog(ctx).WorkspaceService.MigrateWorkspace(ctx, workspaceID, groupID)
	if err != nil {
		return nil, err
	}

	payload := WorkspaceMutationPayload{ClientMutationID: input.ClientMutationID, Workspace: workspace, Problems: []Problem{}}
	return &WorkspaceMutationPayloadResolver{WorkspaceMutationPayload: payload}, nil
}

/* Workspace Subscriptions */

// WorkspaceEventResolver resolves a workspace event
type WorkspaceEventResolver struct {
	event *workspace.Event
}

// Action resolves the event action
func (r *WorkspaceEventResolver) Action() string {
	return r.event.Action
}

// Workspace resolver
func (r *WorkspaceEventResolver) Workspace() *WorkspaceResolver {
	return &WorkspaceResolver{workspace: &r.event.Workspace}
}

// WorkspaceSubscriptionInput is the input for subscribing to workspace events
type WorkspaceSubscriptionInput struct {
	WorkspacePath *string // DEPRECATED: use WorkspaceID instead with a TRN
	WorkspaceID   *string
}

func (r RootResolver) workspaceEventsSubscription(ctx context.Context, input *WorkspaceSubscriptionInput) (<-chan *WorkspaceEventResolver, error) {
	workspaceID, err := toModelID(ctx, input.WorkspacePath, input.WorkspaceID, types.WorkspaceModelType)
	if err != nil {
		return nil, err
	}

	events, err := getServiceCatalog(ctx).WorkspaceService.SubscribeToWorkspaceEvents(ctx, &workspace.EventSubscriptionOptions{
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, err
	}

	outgoing := make(chan *WorkspaceEventResolver)

	go func() {
		for event := range events {
			select {
			case <-ctx.Done():
			case outgoing <- &WorkspaceEventResolver{event: event}:
			}
		}

		close(outgoing)
	}()

	return outgoing, nil
}

/* Workspace loader */

const workspaceLoaderKey = "workspace"

// RegisterWorkspaceLoader registers a workspace loader function
func RegisterWorkspaceLoader(collection *loader.Collection) {
	collection.Register(workspaceLoaderKey, workspaceBatchFunc)
}

func loadWorkspace(ctx context.Context, id string) (*models.Workspace, error) {
	ldr, err := loader.Extract(ctx, workspaceLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	ws, ok := data.(models.Workspace)
	if !ok {
		return nil, errors.New("Wrong type")
	}

	return &ws, nil
}

func workspaceBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	workspaces, err := getServiceCatalog(ctx).WorkspaceService.GetWorkspacesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range workspaces {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}
