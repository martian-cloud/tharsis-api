package resolver

import (
	"context"
	"strconv"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/namespacemembership"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/serviceaccount"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"

	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
)

/* ServiceAccount Query Resolvers */

// JWTClaim represents a claim that must be present in the JWT token
type JWTClaim struct {
	Name  string
	Value string
}

// OIDCTrustPolicy specifies the trust policies for a service account
type OIDCTrustPolicy struct {
	Issuer          string
	BoundClaimsType *models.BoundClaimsType
	BoundClaims     []JWTClaim
}

// ServiceAccountsConnectionQueryArgs are used to query a serviceAccount connection
type ServiceAccountsConnectionQueryArgs struct {
	ConnectionQueryArgs
	IncludeInherited *bool
	Search           *string
}

// ServiceAccountQueryArgs are used to query a single serviceAccount
// DEPRECATED: use node query instead
type ServiceAccountQueryArgs struct {
	ID string
}

// ServiceAccountEdgeResolver resolves serviceAccount edges
type ServiceAccountEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *ServiceAccountEdgeResolver) Cursor() (string, error) {
	serviceAccount, ok := r.edge.Node.(models.ServiceAccount)
	if !ok {
		return "", errors.New("Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&serviceAccount)
	return *cursor, err
}

// Node returns a serviceAccount node
func (r *ServiceAccountEdgeResolver) Node() (*ServiceAccountResolver, error) {
	serviceAccount, ok := r.edge.Node.(models.ServiceAccount)
	if !ok {
		return nil, errors.New("Failed to convert node type")
	}

	return &ServiceAccountResolver{serviceAccount: &serviceAccount}, nil
}

// ServiceAccountConnectionResolver resolves a serviceAccount connection
type ServiceAccountConnectionResolver struct {
	connection Connection
}

// NewServiceAccountConnectionResolver creates a new ServiceAccountConnectionResolver
func NewServiceAccountConnectionResolver(ctx context.Context, input *serviceaccount.GetServiceAccountsInput) (*ServiceAccountConnectionResolver, error) {
	saService := getServiceCatalog(ctx).ServiceAccountService

	result, err := saService.GetServiceAccounts(ctx, input)
	if err != nil {
		return nil, err
	}

	serviceAccounts := result.ServiceAccounts

	// Create edges
	edges := make([]Edge, len(serviceAccounts))
	for i, serviceAccount := range serviceAccounts {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: serviceAccount}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(serviceAccounts) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&serviceAccounts[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&serviceAccounts[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &ServiceAccountConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *ServiceAccountConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *ServiceAccountConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *ServiceAccountConnectionResolver) Edges() *[]*ServiceAccountEdgeResolver {
	resolvers := make([]*ServiceAccountEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &ServiceAccountEdgeResolver{edge: edge}
	}
	return &resolvers
}

// ServiceAccountResolver resolves a serviceAccount resource
type ServiceAccountResolver struct {
	serviceAccount *models.ServiceAccount
}

// ID resolver
func (r *ServiceAccountResolver) ID() graphql.ID {
	return graphql.ID(r.serviceAccount.GetGlobalID())
}

// GroupPath resolver
func (r *ServiceAccountResolver) GroupPath() string {
	return r.serviceAccount.GetGroupPath()
}

// ResourcePath resolver
func (r *ServiceAccountResolver) ResourcePath() string {
	return r.serviceAccount.GetResourcePath()
}

// Name resolver
func (r *ServiceAccountResolver) Name() string {
	return r.serviceAccount.Name
}

// Description resolver
func (r *ServiceAccountResolver) Description() string {
	return r.serviceAccount.Description
}

// Metadata resolver
func (r *ServiceAccountResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.serviceAccount.Metadata}
}

// Group resolver
func (r *ServiceAccountResolver) Group(ctx context.Context) (*GroupResolver, error) {
	group, err := loadGroup(ctx, r.serviceAccount.GroupID)
	if err != nil {
		return nil, err
	}

	return &GroupResolver{group: group}, nil
}

// CreatedBy resolver
func (r *ServiceAccountResolver) CreatedBy() string {
	return r.serviceAccount.CreatedBy
}

// NamespaceMemberships resolver
func (r *ServiceAccountResolver) NamespaceMemberships(ctx context.Context,
	args *ConnectionQueryArgs,
) (*NamespaceMembershipConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := namespacemembership.GetNamespaceMembershipsForSubjectInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		ServiceAccount:    r.serviceAccount,
	}

	if args.Sort != nil {
		sort := db.NamespaceMembershipSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewNamespaceMembershipConnectionResolver(ctx, &input)
}

// OIDCTrustPolicies resolver
func (r *ServiceAccountResolver) OIDCTrustPolicies() []OIDCTrustPolicy {
	policies := []OIDCTrustPolicy{}
	for _, p := range r.serviceAccount.OIDCTrustPolicies {
		p := p
		policy := OIDCTrustPolicy{
			Issuer:          p.Issuer,
			BoundClaimsType: &p.BoundClaimsType,
			BoundClaims:     []JWTClaim{},
		}
		for k, v := range p.BoundClaims {
			policy.BoundClaims = append(policy.BoundClaims, JWTClaim{
				Name:  k,
				Value: v,
			})
		}
		policies = append(policies, policy)
	}
	return policies
}

// ActivityEvents resolver
func (r *ServiceAccountResolver) ActivityEvents(ctx context.Context,
	args *ActivityEventConnectionQueryArgs,
) (*ActivityEventConnectionResolver, error) {
	input, err := getActivityEventsInputFromQueryArgs(ctx, args)
	if err != nil {
		// error is already a Tharsis error
		return nil, err
	}

	// Need to filter to this service account.
	input.ServiceAccountID = &r.serviceAccount.Metadata.ID

	return NewActivityEventConnectionResolver(ctx, input)
}

// DEPRECATED: use node query instead
func serviceAccountQuery(ctx context.Context, args *ServiceAccountQueryArgs) (*ServiceAccountResolver, error) {
	model, err := getServiceCatalog(ctx).FetchModel(ctx, args.ID)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}

	serviceAccount, ok := model.(*models.ServiceAccount)
	if !ok {
		return nil, errors.New("expected service account type, got %T", model)
	}

	return &ServiceAccountResolver{serviceAccount: serviceAccount}, nil
}

/* ServiceAccount Mutation Resolvers */

// ServiceAccountMutationPayload is the response payload for a serviceAccount mutation
type ServiceAccountMutationPayload struct {
	ClientMutationID *string
	ServiceAccount   *models.ServiceAccount
	Problems         []Problem
}

// ServiceAccountMutationPayloadResolver resolves a ServiceAccountMutationPayload
type ServiceAccountMutationPayloadResolver struct {
	ServiceAccountMutationPayload
}

// ServiceAccount field resolver
func (r *ServiceAccountMutationPayloadResolver) ServiceAccount() *ServiceAccountResolver {
	if r.ServiceAccountMutationPayload.ServiceAccount == nil {
		return nil
	}
	return &ServiceAccountResolver{serviceAccount: r.ServiceAccountMutationPayload.ServiceAccount}
}

// CreateServiceAccountInput contains the input for creating a new serviceAccount
type CreateServiceAccountInput struct {
	ClientMutationID  *string
	Name              string
	Description       string
	GroupID           *string
	GroupPath         *string // DEPRECATED: use groupID instead with a TRN
	OIDCTrustPolicies []OIDCTrustPolicy
}

// UpdateServiceAccountInput contains the input for updating a serviceAccount
type UpdateServiceAccountInput struct {
	ClientMutationID  *string
	ID                string
	Metadata          *MetadataInput
	Description       string
	OIDCTrustPolicies []OIDCTrustPolicy
}

// DeleteServiceAccountInput contains the input for deleting a serviceAccount
type DeleteServiceAccountInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	ID               string
}

func handleServiceAccountMutationProblem(e error, clientMutationID *string) (*ServiceAccountMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}

	payload := ServiceAccountMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &ServiceAccountMutationPayloadResolver{ServiceAccountMutationPayload: payload}, nil
}

func createServiceAccountMutation(ctx context.Context, input *CreateServiceAccountInput) (*ServiceAccountMutationPayloadResolver, error) {
	groupID, err := toModelID(ctx, input.GroupPath, input.GroupID, types.GroupModelType)
	if err != nil {
		return nil, err
	}

	oidcTrustPolicies, err := convertOIDCTrustPolicies(input.OIDCTrustPolicies)
	if err != nil {
		return nil, err
	}

	serviceAccountCreateOptions := models.ServiceAccount{
		Name:              input.Name,
		Description:       input.Description,
		GroupID:           groupID,
		OIDCTrustPolicies: oidcTrustPolicies,
	}

	createdServiceAccount, err := getServiceCatalog(ctx).ServiceAccountService.CreateServiceAccount(ctx, &serviceAccountCreateOptions)
	if err != nil {
		return nil, err
	}

	payload := ServiceAccountMutationPayload{ClientMutationID: input.ClientMutationID, ServiceAccount: createdServiceAccount, Problems: []Problem{}}
	return &ServiceAccountMutationPayloadResolver{ServiceAccountMutationPayload: payload}, nil
}

func updateServiceAccountMutation(ctx context.Context, input *UpdateServiceAccountInput) (*ServiceAccountMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	id, err := serviceCatalog.FetchModelID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	serviceAccount, err := serviceCatalog.ServiceAccountService.GetServiceAccountByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, cErr := strconv.Atoi(input.Metadata.Version)
		if cErr != nil {
			return nil, cErr
		}

		serviceAccount.Metadata.Version = v
	}

	// Update fields
	serviceAccount.Description = input.Description
	serviceAccount.OIDCTrustPolicies, err = convertOIDCTrustPolicies(input.OIDCTrustPolicies)
	if err != nil {
		return nil, err
	}

	serviceAccount, err = serviceCatalog.ServiceAccountService.UpdateServiceAccount(ctx, serviceAccount)
	if err != nil {
		return nil, err
	}

	payload := ServiceAccountMutationPayload{ClientMutationID: input.ClientMutationID, ServiceAccount: serviceAccount, Problems: []Problem{}}
	return &ServiceAccountMutationPayloadResolver{ServiceAccountMutationPayload: payload}, nil
}

func deleteServiceAccountMutation(ctx context.Context, input *DeleteServiceAccountInput) (*ServiceAccountMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	id, err := serviceCatalog.FetchModelID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	serviceAccount, err := serviceCatalog.ServiceAccountService.GetServiceAccountByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, err := strconv.Atoi(input.Metadata.Version)
		if err != nil {
			return nil, err
		}

		serviceAccount.Metadata.Version = v
	}

	if err := serviceCatalog.ServiceAccountService.DeleteServiceAccount(ctx, serviceAccount); err != nil {
		return nil, err
	}

	payload := ServiceAccountMutationPayload{ClientMutationID: input.ClientMutationID, ServiceAccount: serviceAccount, Problems: []Problem{}}
	return &ServiceAccountMutationPayloadResolver{ServiceAccountMutationPayload: payload}, nil
}

func convertOIDCTrustPolicies(src []OIDCTrustPolicy) ([]models.OIDCTrustPolicy, error) {
	policies := []models.OIDCTrustPolicy{}
	for _, p := range src {
		boundClaimsType := models.BoundClaimsTypeString
		if p.BoundClaimsType != nil {
			boundClaimsType = *p.BoundClaimsType
		}

		policy := models.OIDCTrustPolicy{
			Issuer:          p.Issuer,
			BoundClaimsType: boundClaimsType,
			BoundClaims:     map[string]string{},
		}

		for _, claim := range p.BoundClaims {
			if _, ok := policy.BoundClaims[claim.Name]; ok {
				return nil,
					errors.New(
						"Claim with name %s can only be defined once for each trust policy", claim.Name,
						errors.WithErrorCode(errors.EInvalid),
					)
			}
			policy.BoundClaims[claim.Name] = claim.Value
		}

		policies = append(policies, policy)
	}
	return policies, nil
}

/* ServiceAccount loader */

const serviceAccountLoaderKey = "serviceAccount"

// RegisterServiceAccountLoader registers a serviceAccount loader function
func RegisterServiceAccountLoader(collection *loader.Collection) {
	collection.Register(serviceAccountLoaderKey, serviceAccountBatchFunc)
}

func loadServiceAccount(ctx context.Context, id string) (*models.ServiceAccount, error) {
	ldr, err := loader.Extract(ctx, serviceAccountLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	serviceAccount, ok := data.(models.ServiceAccount)
	if !ok {
		return nil, errors.New("Wrong type")
	}

	return &serviceAccount, nil
}

func serviceAccountBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	serviceAccounts, err := getServiceCatalog(ctx).ServiceAccountService.GetServiceAccountsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range serviceAccounts {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}

/* Service account create token mutation resolvers */

// ServiceAccountCreateTokenInput contains the input for the service account create token mutation.
type ServiceAccountCreateTokenInput struct {
	ClientMutationID   *string
	ServiceAccountID   *string
	ServiceAccountPath *string // DEPRECATED: use ServiceAccountID instead with a TRN
	Token              string
}

// ServiceAccountCreateTokenPayload is the response payload for the service account create token mutation
type ServiceAccountCreateTokenPayload struct {
	ClientMutationID *string
	Token            *string
	ExpiresIn        *int32
	Problems         []Problem
}

func serviceAccountCreateTokenMutation(ctx context.Context,
	input *ServiceAccountCreateTokenInput,
) (*ServiceAccountCreateTokenPayload, error) {
	var serviceAccountValue string
	switch {
	case input.ServiceAccountPath != nil && input.ServiceAccountID != nil:
		return nil, errors.New("cannot specify both serviceAccountID and serviceAccountPath", errors.WithErrorCode(errors.EInvalid))
	case input.ServiceAccountPath != nil:
		serviceAccountValue = types.ServiceAccountModelType.BuildTRN(*input.ServiceAccountPath)
	case input.ServiceAccountID != nil:
		serviceAccountValue = *input.ServiceAccountID
	default:
		return nil, errors.New("either serviceAccountID or serviceAccountPath must be specified", errors.WithErrorCode(errors.EInvalid))
	}

	resp, err := getServiceCatalog(ctx).ServiceAccountService.CreateToken(ctx, &serviceaccount.CreateTokenInput{
		ServiceAccountPublicID: serviceAccountValue,
		Token:                  []byte(input.Token),
	})
	if err != nil {
		return nil, err
	}
	// resp cannot be nil when err is nil

	stringToken := string(resp.Token)
	payload := ServiceAccountCreateTokenPayload{Token: &stringToken, ExpiresIn: &resp.ExpiresIn, Problems: []Problem{}}
	return &payload, nil
}

func handleServiceAccountCreateTokenProblem(e error, clientMutationID *string) (*ServiceAccountCreateTokenPayload, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := ServiceAccountCreateTokenPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &payload, nil
}
