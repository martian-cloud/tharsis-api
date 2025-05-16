// Package resolver package
package resolver

import (
	"context"
	"encoding/json"
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"

	"github.com/aws/smithy-go/ptr"
	graphql "github.com/graph-gophers/graphql-go"
)

/* ActivityEvent Query Resolvers */

// ActivityEventConnectionQueryArgs are used to query an activity event connection
type ActivityEventConnectionQueryArgs struct {
	ConnectionQueryArgs
	UserID             *string
	ServiceAccountID   *string
	Username           *string // DEPRECATED: use UserID instead with a TRN
	ServiceAccountPath *string // DEPRECATED: use ServiceAccountID instead with a TRN
	NamespacePath      *string
	IncludeNested      *bool
	TimeRangeStart     *graphql.Time
	TimeRangeEnd       *graphql.Time
	Actions            *[]models.ActivityEventAction
	TargetTypes        *[]models.ActivityEventTargetType
}

// ActivityEventEdgeResolver resolves activity event edges
type ActivityEventEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *ActivityEventEdgeResolver) Cursor() (string, error) {
	activityEvent, ok := r.edge.Node.(models.ActivityEvent)
	if !ok {
		return "", errors.New("Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&activityEvent)
	return *cursor, err
}

// Node returns an activity event node
func (r *ActivityEventEdgeResolver) Node() (*ActivityEventResolver, error) {
	activityEvent, ok := r.edge.Node.(models.ActivityEvent)
	if !ok {
		return nil, errors.New("Failed to convert node type")
	}

	return &ActivityEventResolver{activityEvent: &activityEvent}, nil
}

// ActivityEventConnectionResolver resolves an activity event connection
type ActivityEventConnectionResolver struct {
	connection Connection
}

// NewActivityEventConnectionResolver creates a new ActivityEventConnectionResolver
func NewActivityEventConnectionResolver(ctx context.Context,
	input *activityevent.GetActivityEventsInput,
) (*ActivityEventConnectionResolver, error) {
	activityService := getServiceCatalog(ctx).ActivityEventService

	result, err := activityService.GetActivityEvents(ctx, input)
	if err != nil {
		return nil, err
	}

	activityEvents := result.ActivityEvents

	// Create edges
	edges := make([]Edge, len(activityEvents))
	for i, activityEvent := range activityEvents {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: activityEvent}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(activityEvents) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&activityEvents[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&activityEvents[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &ActivityEventConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *ActivityEventConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *ActivityEventConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *ActivityEventConnectionResolver) Edges() *[]*ActivityEventEdgeResolver {
	resolvers := make([]*ActivityEventEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &ActivityEventEdgeResolver{edge: edge}
	}
	return &resolvers
}

// ActivityEventInitiatorResolver resolves the Initiator union type
type ActivityEventInitiatorResolver struct {
	result interface{}
}

// Make sure these stay consistent with the union type in activityevent.graphql:

// ToServiceAccount resolves service account initiator types
func (r *ActivityEventInitiatorResolver) ToServiceAccount() (*ServiceAccountResolver, bool) {
	res, ok := r.result.(*ServiceAccountResolver)
	return res, ok
}

// ToUser resolves user initiator types
func (r *ActivityEventInitiatorResolver) ToUser() (*UserResolver, bool) {
	res, ok := r.result.(*UserResolver)
	return res, ok
}

// ActivityEventAddTeamMemberPayloadResolver is a custom payload resolver
type ActivityEventAddTeamMemberPayloadResolver struct {
	payload *models.ActivityEventAddTeamMemberPayload
}

// User resolver
func (r *ActivityEventAddTeamMemberPayloadResolver) User(ctx context.Context) (*UserResolver, error) {
	user, err := loadUser(ctx, *r.payload.UserID)
	if err != nil {
		// Return nil if user is not found since user may have been deleted
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}
	return &UserResolver{user: user}, nil
}

// Maintainer resolver
func (r *ActivityEventAddTeamMemberPayloadResolver) Maintainer() bool {
	return r.payload.Maintainer
}

// ActivityEventRemoveTeamMemberPayloadResolver is a custom payload resolver
type ActivityEventRemoveTeamMemberPayloadResolver struct {
	payload *models.ActivityEventRemoveTeamMemberPayload
}

// User resolver
func (r *ActivityEventRemoveTeamMemberPayloadResolver) User(ctx context.Context) (*UserResolver, error) {
	user, err := loadUser(ctx, *r.payload.UserID)
	if err != nil {
		// Return nil if user is not found since user may have been deleted
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}
	return &UserResolver{user: user}, nil
}

// ActivityEventUpdateTeamMemberPayloadResolver is a custom payload resolver
type ActivityEventUpdateTeamMemberPayloadResolver struct {
	payload *models.ActivityEventUpdateTeamMemberPayload
}

// User resolver
func (r *ActivityEventUpdateTeamMemberPayloadResolver) User(ctx context.Context) (*UserResolver, error) {
	user, err := loadUser(ctx, *r.payload.UserID)
	if err != nil {
		// Return nil if user is not found since user may have been deleted
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}
	return &UserResolver{user: user}, nil
}

// Maintainer resolver
func (r *ActivityEventUpdateTeamMemberPayloadResolver) Maintainer() bool {
	return r.payload.Maintainer
}

// ActivityEventPayloadResolver resolves the Payload union type
type ActivityEventPayloadResolver struct {
	result interface{}
}

// ToActivityEventCreateNamespaceMembershipPayload resolves the custom payload for creating a namespace membership.
func (r *ActivityEventPayloadResolver) ToActivityEventCreateNamespaceMembershipPayload() (*ActivityEventCreateNamespaceMembershipPayloadResolver, bool) {
	res, ok := r.result.(*ActivityEventCreateNamespaceMembershipPayloadResolver)
	return res, ok
}

// ToActivityEventUpdateNamespaceMembershipPayload resolves the custom payload for updating a namespace membership.
func (r *ActivityEventPayloadResolver) ToActivityEventUpdateNamespaceMembershipPayload() (*models.ActivityEventUpdateNamespaceMembershipPayload, bool) {
	res, ok := r.result.(*models.ActivityEventUpdateNamespaceMembershipPayload)
	return res, ok
}

// ToActivityEventRemoveNamespaceMembershipPayload resolves the custom payload for removing a namespace membership.
func (r *ActivityEventPayloadResolver) ToActivityEventRemoveNamespaceMembershipPayload() (*ActivityEventRemoveNamespaceMembershipPayloadResolver, bool) {
	res, ok := r.result.(*ActivityEventRemoveNamespaceMembershipPayloadResolver)
	return res, ok
}

// ToActivityEventDeleteChildResourcePayload resolves the custom payload for deleting (from a group) a child resource.
func (r *ActivityEventPayloadResolver) ToActivityEventDeleteChildResourcePayload() (*models.ActivityEventDeleteChildResourcePayload, bool) {
	res, ok := r.result.(*models.ActivityEventDeleteChildResourcePayload)
	return res, ok
}

// ToActivityEventAddTeamMemberPayload resolver
func (r *ActivityEventPayloadResolver) ToActivityEventAddTeamMemberPayload() (*ActivityEventAddTeamMemberPayloadResolver, bool) {
	res, ok := r.result.(*ActivityEventAddTeamMemberPayloadResolver)
	return res, ok
}

// ToActivityEventRemoveTeamMemberPayload resolver
func (r *ActivityEventPayloadResolver) ToActivityEventRemoveTeamMemberPayload() (*ActivityEventRemoveTeamMemberPayloadResolver, bool) {
	res, ok := r.result.(*ActivityEventRemoveTeamMemberPayloadResolver)
	return res, ok
}

// ToActivityEventUpdateTeamMemberPayload resolver
func (r *ActivityEventPayloadResolver) ToActivityEventUpdateTeamMemberPayload() (*ActivityEventUpdateTeamMemberPayloadResolver, bool) {
	res, ok := r.result.(*ActivityEventUpdateTeamMemberPayloadResolver)
	return res, ok
}

// ToActivityEventMigrateGroupPayload resolver
func (r *ActivityEventPayloadResolver) ToActivityEventMigrateGroupPayload() (*ActivityEventMigrateGroupPayloadResolver, bool) {
	res, ok := r.result.(*ActivityEventMigrateGroupPayloadResolver)
	return res, ok
}

// ToActivityEventMigrateWorkspacePayload resolver
func (r *ActivityEventPayloadResolver) ToActivityEventMigrateWorkspacePayload() (*ActivityEventMigrateWorkspacePayloadResolver, bool) {
	res, ok := r.result.(*ActivityEventMigrateWorkspacePayloadResolver)
	return res, ok
}

// ToActivityEventMoveManagedIdentityPayload resolver
func (r *ActivityEventPayloadResolver) ToActivityEventMoveManagedIdentityPayload() (*ActivityEventMoveManagedIdentityPayloadResolver, bool) {
	res, ok := r.result.(*ActivityEventMoveManagedIdentityPayloadResolver)
	return res, ok
}

// ActivityEventResolver resolves an activity event resource
type ActivityEventResolver struct {
	activityEvent *models.ActivityEvent
}

// ID resolver
func (r *ActivityEventResolver) ID() graphql.ID {
	return graphql.ID(r.activityEvent.GetGlobalID())
}

// Metadata resolver
func (r *ActivityEventResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.activityEvent.Metadata}
}

// Initiator resolver
func (r *ActivityEventResolver) Initiator(ctx context.Context) (*ActivityEventInitiatorResolver, error) {
	// Query for initiator based on type
	// (sorted by type)

	switch {
	case r.activityEvent.UserID != nil:
		// Use resource loader to get service account
		user, err := loadUser(ctx, *r.activityEvent.UserID)
		if err != nil {
			return nil, err
		}
		return &ActivityEventInitiatorResolver{result: &UserResolver{user: user}}, nil
	case r.activityEvent.ServiceAccountID != nil:
		// Use resource loader to get service account
		serviceAccount, err := loadServiceAccount(ctx, *r.activityEvent.ServiceAccountID)
		if err != nil {
			return nil, err
		}
		return &ActivityEventInitiatorResolver{result: &ServiceAccountResolver{serviceAccount: serviceAccount}}, nil
	default:
		return nil, fmt.Errorf("activity event must have either a user ID or a service account ID")
	}
}

// NamespacePath resolver
func (r *ActivityEventResolver) NamespacePath() *string {
	return r.activityEvent.NamespacePath
}

// Action resolver
func (r *ActivityEventResolver) Action() string {
	return string(r.activityEvent.Action)
}

// Target resolver
func (r *ActivityEventResolver) Target(ctx context.Context) (*NodeResolver, error) {
	// Query for target based on type
	// (sorted by type)

	switch r.activityEvent.TargetType {
	case models.TargetGPGKey:
		// Use resource loader to get GPG key
		gpgKey, err := loadGPGKey(ctx, r.activityEvent.TargetID)
		if err != nil {
			return nil, err
		}
		return &NodeResolver{result: &GPGKeyResolver{gpgKey: gpgKey}}, nil
	case models.TargetGroup:
		// Use resource loader to get group
		group, err := loadGroup(ctx, r.activityEvent.TargetID)
		if err != nil {
			return nil, err
		}
		return &NodeResolver{result: &GroupResolver{group: group}}, nil
	case models.TargetManagedIdentity:
		// Use resource loader to get managed identity
		managedIdentity, err := loadManagedIdentity(ctx, r.activityEvent.TargetID)
		if err != nil {
			return nil, err
		}
		return &NodeResolver{result: &ManagedIdentityResolver{managedIdentity: managedIdentity}}, nil
	case models.TargetManagedIdentityAccessRule:
		// Use resource loader to get managed identity
		rule, err := loadManagedIdentityAccessRule(ctx, r.activityEvent.TargetID)
		if err != nil {
			return nil, err
		}
		return &NodeResolver{result: &ManagedIdentityAccessRuleResolver{rule: rule}}, nil
	case models.TargetNamespaceMembership:
		// Use resource loader to get namespace membership
		namespaceMembership, err := loadNamespaceMembership(ctx, r.activityEvent.TargetID)
		if err != nil {
			return nil, err
		}
		return &NodeResolver{result: &NamespaceMembershipResolver{namespaceMembership: namespaceMembership}}, nil
	case models.TargetRun:
		// Use resource loader to get run
		run, err := loadRun(ctx, r.activityEvent.TargetID)
		if err != nil {
			return nil, err
		}
		return &NodeResolver{result: &RunResolver{run: run}}, nil
	case models.TargetServiceAccount:
		// Use resource loader to get service account
		serviceAccount, err := loadServiceAccount(ctx, r.activityEvent.TargetID)
		if err != nil {
			return nil, err
		}
		return &NodeResolver{result: &ServiceAccountResolver{serviceAccount: serviceAccount}}, nil
	case models.TargetTeam:
		// Use resource loader to get team
		team, err := loadTeam(ctx, r.activityEvent.TargetID)
		if err != nil {
			return nil, err
		}
		return &NodeResolver{result: &TeamResolver{team: team}}, nil
	case models.TargetVariable:
		// Use resource loader to get variable
		variable, err := loadVariable(ctx, r.activityEvent.TargetID)
		if err != nil {
			return nil, err
		}
		return &NodeResolver{result: &NamespaceVariableResolver{variable: variable}}, nil
	case models.TargetWorkspace:
		// Use resource loader to get workspace
		workspace, err := loadWorkspace(ctx, r.activityEvent.TargetID)
		if err != nil {
			return nil, err
		}
		return &NodeResolver{result: &WorkspaceResolver{workspace: workspace}}, nil
	case models.TargetTerraformProvider:
		tfProvider, err := loadTerraformProvider(ctx, r.activityEvent.TargetID)
		if err != nil {
			return nil, err
		}
		return &NodeResolver{result: &TerraformProviderResolver{provider: tfProvider}}, nil
	case models.TargetTerraformProviderVersion:
		tfProviderVersion, err := loadTerraformProviderVersion(ctx, r.activityEvent.TargetID)
		if err != nil {
			return nil, err
		}
		return &NodeResolver{result: &TerraformProviderVersionResolver{providerVersion: tfProviderVersion}}, nil
	case models.TargetTerraformModule:
		tfModule, err := loadTerraformModule(ctx, r.activityEvent.TargetID)
		if err != nil {
			return nil, err
		}
		return &NodeResolver{result: &TerraformModuleResolver{module: tfModule}}, nil
	case models.TargetTerraformModuleVersion:
		tfModuleVersion, err := loadTerraformModuleVersion(ctx, r.activityEvent.TargetID)
		if err != nil {
			return nil, err
		}
		return &NodeResolver{result: &TerraformModuleVersionResolver{moduleVersion: tfModuleVersion}}, nil
	case models.TargetStateVersion:
		stateVersion, err := loadStateVersion(ctx, r.activityEvent.TargetID)
		if err != nil {
			return nil, err
		}
		return &NodeResolver{result: &StateVersionResolver{stateVersion: stateVersion}}, nil
	case models.TargetVCSProvider:
		vcsProvider, err := loadVCSProvider(ctx, r.activityEvent.TargetID)
		if err != nil {
			return nil, err
		}
		return &NodeResolver{result: &VCSProviderResolver{vcsProvider: vcsProvider}}, nil
	case models.TargetRole:
		role, err := loadRole(ctx, r.activityEvent.TargetID)
		if err != nil {
			return nil, err
		}
		return &NodeResolver{result: &RoleResolver{role: role}}, nil
	case models.TargetRunner:
		runner, err := loadRunner(ctx, r.activityEvent.TargetID)
		if err != nil {
			return nil, err
		}
		return &NodeResolver{result: &RunnerResolver{runner: runner}}, nil
	case models.TargetTerraformProviderVersionMirror:
		mirror, err := loadTerraformProviderVersionMirror(ctx, r.activityEvent.TargetID)
		if err != nil {
			return nil, err
		}
		return &NodeResolver{result: &TerraformProviderVersionMirrorResolver{versionMirror: mirror}}, nil
	case models.TargetFederatedRegistry:
		federatedRegistry, err := loadFederatedRegistry(ctx, r.activityEvent.TargetID)
		if err != nil {
			return nil, err
		}
		return &NodeResolver{result: &FederatedRegistryResolver{federatedRegistry: federatedRegistry}}, nil
	default:
		return nil, errors.New("valid TargetType must be specified", errors.WithErrorCode(errors.EInvalid))
	}
}

// TargetType resolver
func (r *ActivityEventResolver) TargetType() models.ActivityEventTargetType {
	return r.activityEvent.TargetType
}

// TargetID resolver
func (r *ActivityEventResolver) TargetID() string {
	return gid.ToGlobalID(types.ActivityEventModelType, r.activityEvent.TargetID)
}

// Payload resolver
func (r *ActivityEventResolver) Payload() (*ActivityEventPayloadResolver, error) {
	if r.activityEvent.Payload != nil {
		switch {

		case r.activityEvent.Action == models.ActionCreateMembership:
			var payload models.ActivityEventCreateNamespaceMembershipPayload
			if err := json.Unmarshal(r.activityEvent.Payload, &payload); err != nil {
				return nil, err
			}
			return &ActivityEventPayloadResolver{
				result: &ActivityEventCreateNamespaceMembershipPayloadResolver{payload: &payload},
			}, nil
		case (r.activityEvent.Action == models.ActionUpdate) &&
			(r.activityEvent.TargetType == models.TargetNamespaceMembership):
			var payload models.ActivityEventUpdateNamespaceMembershipPayload
			if err := json.Unmarshal(r.activityEvent.Payload, &payload); err != nil {
				return nil, err
			}
			return &ActivityEventPayloadResolver{result: &payload}, nil

		case r.activityEvent.Action == models.ActionRemoveMembership:
			var payload models.ActivityEventRemoveNamespaceMembershipPayload
			if err := json.Unmarshal(r.activityEvent.Payload, &payload); err != nil {
				return nil, err
			}
			return &ActivityEventPayloadResolver{
				result: &ActivityEventRemoveNamespaceMembershipPayloadResolver{payload: &payload},
			}, nil

		case r.activityEvent.Action == models.ActionDeleteChildResource:
			var payload models.ActivityEventDeleteChildResourcePayload
			if err := json.Unmarshal(r.activityEvent.Payload, &payload); err != nil {
				return nil, err
			}
			return &ActivityEventPayloadResolver{result: &payload}, nil
		case (r.activityEvent.Action == models.ActionAddMember) &&
			(r.activityEvent.TargetType == models.TargetTeam):
			var payload models.ActivityEventAddTeamMemberPayload
			if err := json.Unmarshal(r.activityEvent.Payload, &payload); err != nil {
				return nil, err
			}
			return &ActivityEventPayloadResolver{result: &ActivityEventAddTeamMemberPayloadResolver{payload: &payload}}, nil
		case (r.activityEvent.Action == models.ActionRemoveMember) &&
			(r.activityEvent.TargetType == models.TargetTeam):
			var payload models.ActivityEventRemoveTeamMemberPayload
			if err := json.Unmarshal(r.activityEvent.Payload, &payload); err != nil {
				return nil, err
			}
			return &ActivityEventPayloadResolver{result: &ActivityEventRemoveTeamMemberPayloadResolver{payload: &payload}}, nil
		case (r.activityEvent.Action == models.ActionUpdateMember) &&
			(r.activityEvent.TargetType == models.TargetTeam):
			var payload models.ActivityEventUpdateTeamMemberPayload
			if err := json.Unmarshal(r.activityEvent.Payload, &payload); err != nil {
				return nil, err
			}
			return &ActivityEventPayloadResolver{result: &ActivityEventUpdateTeamMemberPayloadResolver{payload: &payload}}, nil
		case (r.activityEvent.Action == models.ActionMigrate) &&
			(r.activityEvent.TargetType == models.TargetGroup):
			var payload models.ActivityEventMigrateGroupPayload
			if err := json.Unmarshal(r.activityEvent.Payload, &payload); err != nil {
				return nil, err
			}
			return &ActivityEventPayloadResolver{result: &ActivityEventMigrateGroupPayloadResolver{payload: &payload}}, nil
		case (r.activityEvent.Action == models.ActionMigrate) &&
			(r.activityEvent.TargetType == models.TargetWorkspace):
			var payload models.ActivityEventMigrateWorkspacePayload
			if err := json.Unmarshal(r.activityEvent.Payload, &payload); err != nil {
				return nil, err
			}
			return &ActivityEventPayloadResolver{result: &ActivityEventMigrateWorkspacePayloadResolver{payload: &payload}}, nil
		case (r.activityEvent.Action == models.ActionMigrate) &&
			(r.activityEvent.TargetType == models.TargetManagedIdentity):
			var payload models.ActivityEventMoveManagedIdentityPayload
			if err := json.Unmarshal(r.activityEvent.Payload, &payload); err != nil {
				return nil, err
			}
			return &ActivityEventPayloadResolver{result: &ActivityEventMoveManagedIdentityPayloadResolver{payload: &payload}}, nil
		default:
			return nil, fmt.Errorf("payload supplied without a supported target type and action")

		}
	}
	return nil, nil
}

// ActivityEventCreateNamespaceMembershipPayloadResolver resolves an
// activity event create namespace membership payload resource
type ActivityEventCreateNamespaceMembershipPayloadResolver struct {
	payload *models.ActivityEventCreateNamespaceMembershipPayload
}

// Member resolver
func (r *ActivityEventCreateNamespaceMembershipPayloadResolver) Member(ctx context.Context) (*MemberResolver, error) {
	resolver, err := makeMemberResolver(ctx, r.payload.UserID,
		r.payload.ServiceAccountID,
		r.payload.TeamID)
	if errors.ErrorCode(err) == errors.ENotFound {
		return nil, nil
	}
	return resolver, err
}

// Role resolver
func (r *ActivityEventCreateNamespaceMembershipPayloadResolver) Role() string {
	return r.payload.Role
}

// ActivityEventRemoveNamespaceMembershipPayloadResolver resolves an
// activity event remove namespace membership payload resource
type ActivityEventRemoveNamespaceMembershipPayloadResolver struct {
	payload *models.ActivityEventRemoveNamespaceMembershipPayload
}

// Member resolver
func (r *ActivityEventRemoveNamespaceMembershipPayloadResolver) Member(ctx context.Context) (*MemberResolver, error) {
	resolver, err := makeMemberResolver(ctx, r.payload.UserID,
		r.payload.ServiceAccountID,
		r.payload.TeamID)
	if errors.ErrorCode(err) == errors.ENotFound {
		return nil, nil
	}
	return resolver, err
}

// ActivityEventDeleteChildResourcePayloadResolver resolves an
// activity event delete child resource payload resource
type ActivityEventDeleteChildResourcePayloadResolver struct {
	activityEventDeleteChildResourcePayload *models.ActivityEventDeleteChildResourcePayload
}

// Name resolver
func (r *ActivityEventDeleteChildResourcePayloadResolver) Name() string {
	return r.activityEventDeleteChildResourcePayload.Name
}

// ID resolver
func (r *ActivityEventDeleteChildResourcePayloadResolver) ID() string {
	return r.activityEventDeleteChildResourcePayload.ID
}

// Type resolver
func (r *ActivityEventDeleteChildResourcePayloadResolver) Type() string {
	return r.activityEventDeleteChildResourcePayload.Type
}

// ActivityEventMigrateGroupPayloadResolver resolves an activity event
// migrate group payload resource
type ActivityEventMigrateGroupPayloadResolver struct {
	payload *models.ActivityEventMigrateGroupPayload
}

// ActivityEventMigrateWorkspacePayloadResolver resolves an activity event
// migrate workspace payload resource
type ActivityEventMigrateWorkspacePayloadResolver struct {
	payload *models.ActivityEventMigrateWorkspacePayload
}

// PreviousGroupPath resolver
func (r *ActivityEventMigrateGroupPayloadResolver) PreviousGroupPath() string {
	return r.payload.PreviousGroupPath
}

// PreviousGroupPath resolver (for workspace migration)
func (r *ActivityEventMigrateWorkspacePayloadResolver) PreviousGroupPath() string {
	return r.payload.PreviousGroupPath
}

// ActivityEventMoveManagedIdentityPayloadResolver resolves an activity event
// move managed identity payload resource
type ActivityEventMoveManagedIdentityPayloadResolver struct {
	payload *models.ActivityEventMoveManagedIdentityPayload
}

// PreviousGroupPath resolver
func (r *ActivityEventMoveManagedIdentityPayloadResolver) PreviousGroupPath() string {
	return r.payload.PreviousGroupPath
}

func activityEventsQuery(ctx context.Context, args *ActivityEventConnectionQueryArgs) (*ActivityEventConnectionResolver, error) {
	input, err := getActivityEventsInputFromQueryArgs(ctx, args)
	if err != nil {
		// If needed, the error is already a Tharsis error.
		return nil, err
	}

	// For the top-level activity events query, no changes need to be made to the input struct.

	return NewActivityEventConnectionResolver(ctx, input)
}

// getActivityEventsInputFromQueryArgs is for the convenience of other modules in this package
// Other modules may need to modify the input before creating a resolver.
func getActivityEventsInputFromQueryArgs(ctx context.Context,
	args *ActivityEventConnectionQueryArgs,
) (*activityevent.GetActivityEventsInput, error) {
	if err := args.Validate(); err != nil {
		// if needed, the error is already a Tharsis error
		return nil, err
	}

	input := activityevent.GetActivityEventsInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
	}

	if args.UserID != nil && args.Username != nil {
		return nil, errors.New("cannot specify both userID and username", errors.WithErrorCode(errors.EInvalid))
	}

	if args.ServiceAccountID != nil && args.ServiceAccountPath != nil {
		return nil, errors.New("cannot specify both serviceAccountID and serviceAccountPath", errors.WithErrorCode(errors.EInvalid))
	}

	var userIDToResolve, serviceAccountIDToResolve *string

	if args.UserID != nil {
		userIDToResolve = args.UserID
	} else if args.Username != nil {
		userIDToResolve = ptr.String(types.UserModelType.BuildTRN(*args.Username))
	}

	if args.ServiceAccountID != nil {
		serviceAccountIDToResolve = args.ServiceAccountID
	} else if args.ServiceAccountPath != nil {
		serviceAccountIDToResolve = ptr.String(types.ServiceAccountModelType.BuildTRN(*args.ServiceAccountPath))
	}

	// Resolve the user's ID to pass in as a filter
	if userIDToResolve != nil {
		userID, err := getServiceCatalog(ctx).FetchModelID(ctx, *userIDToResolve)
		if err != nil {
			return nil, err
		}

		input.UserID = &userID
	}

	if serviceAccountIDToResolve != nil {
		serviceAccountID, err := getServiceCatalog(ctx).FetchModelID(ctx, *serviceAccountIDToResolve)
		if err != nil {
			return nil, err
		}

		input.ServiceAccountID = &serviceAccountID
	}

	if args.NamespacePath != nil {
		input.NamespacePath = args.NamespacePath
	}

	if args.IncludeNested != nil {
		input.IncludeNested = *args.IncludeNested
		// otherwise, leave it false
	}

	if args.TimeRangeStart != nil {
		input.TimeRangeStart = &args.TimeRangeStart.Time
	}

	if args.TimeRangeEnd != nil {
		input.TimeRangeEnd = &args.TimeRangeEnd.Time
	}

	if args.Actions != nil {
		input.Actions = *args.Actions
	}

	if args.TargetTypes != nil {
		input.TargetTypes = *args.TargetTypes
	}

	if args.Sort != nil {
		sort := db.ActivityEventSortableField(*args.Sort)
		input.Sort = &sort
	}

	return &input, nil
}

/* ActivityEvent Mutation Resolvers do not exist. */
