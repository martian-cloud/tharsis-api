package resolver

import (
	"context"

	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

/* WorkspaceRoleBinding Query Resolvers */

// WorkspaceRoleBindingResolver resolves a workspaceRoleBinding resource. Binding a role to a
// workspace confers that role's permissions on the workspace's job caller at the workspace's DIRECT
// PARENT namespace — see models.WorkspaceRoleBinding for the full explanation.
type WorkspaceRoleBindingResolver struct {
	workspaceRoleBinding *models.WorkspaceRoleBinding
}

// ID resolver
func (r *WorkspaceRoleBindingResolver) ID() graphql.ID {
	return graphql.ID(r.workspaceRoleBinding.GetGlobalID())
}

// Metadata resolver
func (r *WorkspaceRoleBindingResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.workspaceRoleBinding.Metadata}
}

// CreatedBy resolver
func (r *WorkspaceRoleBindingResolver) CreatedBy() string {
	return r.workspaceRoleBinding.CreatedBy
}

// Workspace resolver
func (r *WorkspaceRoleBindingResolver) Workspace(ctx context.Context) (*WorkspaceResolver, error) {
	ws, err := loadWorkspace(ctx, r.workspaceRoleBinding.WorkspaceID)
	if err != nil {
		return nil, err
	}

	return &WorkspaceResolver{workspace: ws}, nil
}

// Role resolver
func (r *WorkspaceRoleBindingResolver) Role(ctx context.Context) (*RoleResolver, error) {
	role, err := loadRole(ctx, r.workspaceRoleBinding.RoleID)
	if err != nil {
		return nil, err
	}

	return &RoleResolver{role: role}, nil
}

/* WorkspaceRoleBinding loader */

const workspaceRoleBindingByWorkspaceIDLoaderKey = "workspaceRoleBindingByWorkspaceID"

// RegisterWorkspaceRoleBindingByWorkspaceIDLoader registers a workspaceRoleBindingByWorkspaceID
// loader function, keyed by workspace ID rather than by the binding's own ID, since the resolver
// always looks a binding up starting from its workspace.
func RegisterWorkspaceRoleBindingByWorkspaceIDLoader(collection *loader.Collection) {
	collection.Register(workspaceRoleBindingByWorkspaceIDLoaderKey, workspaceRoleBindingByWorkspaceIDBatchFunc)
}

// loadWorkspaceRoleBindingByWorkspaceID returns the binding for a workspace, or nil if the workspace
// has none. A missing entry from the batch is not an error — most workspaces have no binding — so
// ENotFound from the loader is translated to (nil, nil) here rather than propagated.
func loadWorkspaceRoleBindingByWorkspaceID(ctx context.Context, workspaceID string) (*models.WorkspaceRoleBinding, error) {
	ldr, err := loader.Extract(ctx, workspaceRoleBindingByWorkspaceIDLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(workspaceID))()
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}

	binding, ok := data.(models.WorkspaceRoleBinding)
	if !ok {
		return nil, errors.New("Wrong type")
	}

	return &binding, nil
}

func workspaceRoleBindingByWorkspaceIDBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	bindings, err := getServiceCatalog(ctx).WorkspaceService.GetWorkspaceRoleBindingsByWorkspaceIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results, keyed by workspace ID. Workspaces with no binding are simply absent.
	batch := loader.DataBatch{}
	for _, result := range bindings {
		batch[result.WorkspaceID] = result
	}

	return batch, nil
}

/* WorkspaceRoleBinding-by-own-ID loader, used by the activity event resolver */

const workspaceRoleBindingByIDLoaderKey = "workspaceRoleBindingByID"

// RegisterWorkspaceRoleBindingByIDLoader registers a loader keyed by a binding's own ID, distinct
// from RegisterWorkspaceRoleBindingByWorkspaceIDLoader (keyed by workspace ID). The activity event resolver looks
// a binding up starting from the event's TargetID, which IS the binding's own ID — a workspace-keyed
// lookup cannot serve that, since the workspace the binding belonged to isn't known without first
// loading the binding, and a REMOVE event's binding may no longer exist at all.
func RegisterWorkspaceRoleBindingByIDLoader(collection *loader.Collection) {
	collection.Register(workspaceRoleBindingByIDLoaderKey, workspaceRoleBindingByIDBatchFunc)
}

// loadWorkspaceRoleBindingByID returns the binding with the given ID. Unlike
// loadWorkspaceRoleBindingByWorkspaceID (workspace-keyed), a missing entry here IS an error
// (ENotFound) — the activity event resolver translates that into "target no longer exists," the
// same handling as every other target type.
func loadWorkspaceRoleBindingByID(ctx context.Context, id string) (*models.WorkspaceRoleBinding, error) {
	ldr, err := loader.Extract(ctx, workspaceRoleBindingByIDLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	binding, ok := data.(models.WorkspaceRoleBinding)
	if !ok {
		return nil, errors.New("Wrong type")
	}

	return &binding, nil
}

func workspaceRoleBindingByIDBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	bindings, err := getServiceCatalog(ctx).WorkspaceService.GetWorkspaceRoleBindingsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results, keyed by the binding's own ID. A binding whose ID isn't found (e.g. it
	// was later removed) is simply absent, which the loader machinery reports as ENotFound.
	batch := loader.DataBatch{}
	for _, result := range bindings {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}

/* WorkspaceRoleBinding Mutation Resolvers */

// SetWorkspaceRoleBindingInput is the input for creating, changing, or removing a workspace's role
// binding.
type SetWorkspaceRoleBindingInput struct {
	ClientMutationID *string
	WorkspaceID      string
	RoleID           *string
}

// SetWorkspaceRoleBindingPayload is the response payload for the setWorkspaceRoleBinding mutation.
type SetWorkspaceRoleBindingPayload struct {
	ClientMutationID *string
	Workspace        *models.Workspace
	Problems         []Problem
}

// SetWorkspaceRoleBindingPayloadResolver resolves a SetWorkspaceRoleBindingPayload
type SetWorkspaceRoleBindingPayloadResolver struct {
	SetWorkspaceRoleBindingPayload
}

// Workspace field resolver
func (r *SetWorkspaceRoleBindingPayloadResolver) Workspace() *WorkspaceResolver {
	if r.SetWorkspaceRoleBindingPayload.Workspace == nil {
		return nil
	}
	return &WorkspaceResolver{workspace: r.SetWorkspaceRoleBindingPayload.Workspace}
}

func handleSetWorkspaceRoleBindingMutationProblem(e error, clientMutationID *string) (*SetWorkspaceRoleBindingPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := SetWorkspaceRoleBindingPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &SetWorkspaceRoleBindingPayloadResolver{SetWorkspaceRoleBindingPayload: payload}, nil
}

func setWorkspaceRoleBindingMutation(ctx context.Context, input *SetWorkspaceRoleBindingInput) (*SetWorkspaceRoleBindingPayloadResolver, error) {
	workspaceID, err := toModelID(ctx, nil, &input.WorkspaceID, types.WorkspaceModelType)
	if err != nil {
		return nil, err
	}

	var roleID *string
	if input.RoleID != nil {
		resolvedRoleID, rErr := toModelID(ctx, nil, input.RoleID, types.RoleModelType)
		if rErr != nil {
			return nil, rErr
		}
		roleID = &resolvedRoleID
	}

	_, err = getServiceCatalog(ctx).WorkspaceService.SetWorkspaceRoleBinding(ctx, &workspace.SetWorkspaceRoleBindingInput{
		WorkspaceID: workspaceID,
		RoleID:      roleID,
	})
	if err != nil {
		return nil, err
	}

	ws, err := getServiceCatalog(ctx).WorkspaceService.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	payload := SetWorkspaceRoleBindingPayload{ClientMutationID: input.ClientMutationID, Workspace: ws, Problems: []Problem{}}
	return &SetWorkspaceRoleBindingPayloadResolver{SetWorkspaceRoleBindingPayload: payload}, nil
}
