package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

// group represents a Tharsis group.
type group struct {
	GroupID     string   `json:"group_id" jsonschema:"Unique identifier for this group"`
	TRN         string   `json:"trn" jsonschema:"Tharsis Resource Name (e.g. trn:group:parent/group-name)"`
	Name        string   `json:"name" jsonschema:"Group name (last segment of full path)"`
	FullPath    string   `json:"full_path" jsonschema:"Complete path including parent groups (e.g. org/team/sub-team)"`
	Description string   `json:"description" jsonschema:"Human-readable description of this group's purpose"`
	ParentID    string   `json:"parent_id" jsonschema:"ID of the parent group (empty for root groups)"`
	CreatedBy   string   `json:"created_by" jsonschema:"Email address of the user who created this group"`
	RunnerTags  []string `json:"runner_tags,omitempty" jsonschema:"Tags used to select which runner agents can execute jobs"`
}

// getGroupInput defines the parameters for retrieving a group.
type getGroupInput struct {
	ID string `json:"id" jsonschema:"required,Group ID or TRN"`
}

// getGroupOutput wraps the group response.
type getGroupOutput struct {
	Group group `json:"group" jsonschema:"The group configuration"`
}

// GetGroup returns an MCP tool for retrieving group configuration.
func GetGroup(tc *ToolContext) (mcp.Tool, mcp.ToolHandlerFor[getGroupInput, getGroupOutput]) {
	tool := mcp.Tool{
		Name:        "get_group",
		Description: "Retrieve group configuration and settings. Groups organize workspaces and can be nested hierarchically.",
		Annotations: &mcp.ToolAnnotations{
			Title:        "Get Group",
			ReadOnlyHint: true,
		},
	}

	handler := func(ctx context.Context, _ *mcp.CallToolRequest, input getGroupInput) (*mcp.CallToolResult, getGroupOutput, error) {
		fetchedModel, err := tc.servicesCatalog.FetchModel(ctx, input.ID)
		if err != nil {
			return nil, getGroupOutput{}, WrapMCPToolError(err, "failed to resolve group %q", input.ID)
		}

		g, ok := fetchedModel.(*models.Group)
		if !ok {
			return nil, getGroupOutput{}, NewMCPToolError("group with id %s not found", input.ID)
		}

		return nil, getGroupOutput{
			Group: group{
				GroupID:     g.GetGlobalID(),
				TRN:         g.Metadata.TRN,
				Name:        g.Name,
				FullPath:    g.FullPath,
				Description: g.Description,
				ParentID:    gid.ToGlobalID(types.GroupModelType, g.ParentID),
				CreatedBy:   g.CreatedBy,
				RunnerTags:  g.RunnerTags,
			},
		}, nil
	}

	return tool, handler
}

// managedIdentity represents a Tharsis managed identity.
type managedIdentity struct {
	ManagedIdentityID string `json:"managed_identity_id" jsonschema:"Unique identifier for this managed identity"`
	TRN               string `json:"trn" jsonschema:"Tharsis Resource Name (e.g. trn:managed_identity:group/identity-name)"`
	Name              string `json:"name" jsonschema:"Managed identity name"`
	Description       string `json:"description" jsonschema:"Human-readable description"`
	Type              string `json:"type" jsonschema:"Identity type (aws_federated, azure_federated, tharsis_federated, kubernetes_federated)"`
	GroupID           string `json:"group_id" jsonschema:"ID of the parent group"`
	CreatedBy         string `json:"created_by" jsonschema:"Email address of the creator"`
	IsAlias           bool   `json:"is_alias" jsonschema:"True if this is an alias of another managed identity"`
}

// getManagedIdentityInput defines the parameters for retrieving a managed identity.
type getManagedIdentityInput struct {
	ID string `json:"id" jsonschema:"required,Managed identity ID or TRN (e.g. Ul8yZ... or trn:managed_identity:group/identity-name)"`
}

// getManagedIdentityOutput wraps the managed identity response.
type getManagedIdentityOutput struct {
	ManagedIdentity managedIdentity `json:"managed_identity" jsonschema:"The managed identity configuration"`
}

// GetManagedIdentity returns an MCP tool for retrieving a managed identity.
func GetManagedIdentity(tc *ToolContext) (mcp.Tool, mcp.ToolHandlerFor[getManagedIdentityInput, getManagedIdentityOutput]) {
	tool := mcp.Tool{
		Name:        "get_managed_identity",
		Description: "Retrieve a managed identity's configuration. Managed identities provide federated access to cloud providers for workspaces.",
		Annotations: &mcp.ToolAnnotations{
			Title:        "Get Managed Identity",
			ReadOnlyHint: true,
		},
	}

	handler := func(ctx context.Context, _ *mcp.CallToolRequest, input getManagedIdentityInput) (*mcp.CallToolResult, getManagedIdentityOutput, error) {
		fetchedModel, err := tc.servicesCatalog.FetchModel(ctx, input.ID)
		if err != nil {
			return nil, getManagedIdentityOutput{}, WrapMCPToolError(err, "failed to resolve managed identity %q", input.ID)
		}

		mi, ok := fetchedModel.(*models.ManagedIdentity)
		if !ok {
			return nil, getManagedIdentityOutput{}, NewMCPToolError("managed identity with id %s not found", input.ID)
		}

		return nil, getManagedIdentityOutput{
			ManagedIdentity: managedIdentity{
				ManagedIdentityID: mi.GetGlobalID(),
				TRN:               mi.Metadata.TRN,
				Name:              mi.Name,
				Description:       mi.Description,
				Type:              string(mi.Type),
				GroupID:           gid.ToGlobalID(types.GroupModelType, mi.GroupID),
				CreatedBy:         mi.CreatedBy,
				IsAlias:           mi.AliasSourceID != nil,
			},
		}, nil
	}

	return tool, handler
}

// oidcTrustPolicy represents an OIDC trust policy on a service account.
type oidcTrustPolicy struct {
	Issuer          string            `json:"issuer" jsonschema:"OIDC issuer URL"`
	BoundClaimsType string            `json:"bound_claims_type" jsonschema:"How bound claims are matched (string or glob)"`
	BoundClaims     map[string]string `json:"bound_claims" jsonschema:"Claims that must match for authentication"`
}

// serviceAccount represents a Tharsis service account.
type serviceAccount struct {
	ServiceAccountID         string            `json:"service_account_id" jsonschema:"Unique identifier for this service account"`
	TRN                      string            `json:"trn" jsonschema:"Tharsis Resource Name (e.g. trn:service_account:group/account-name)"`
	Name                     string            `json:"name" jsonschema:"Service account name"`
	Description              string            `json:"description" jsonschema:"Human-readable description"`
	GroupID                  string            `json:"group_id" jsonschema:"ID of the parent group"`
	CreatedBy                string            `json:"created_by" jsonschema:"Email address of the creator"`
	ClientCredentialsEnabled bool              `json:"client_credentials_enabled" jsonschema:"True if client credentials authentication is configured"`
	OIDCTrustPolicies        []oidcTrustPolicy `json:"oidc_trust_policies,omitempty" jsonschema:"OIDC trust policies for federated authentication"`
}

// getServiceAccountInput defines the parameters for retrieving a service account.
type getServiceAccountInput struct {
	ID string `json:"id" jsonschema:"required,Service account ID or TRN (e.g. Ul8yZ... or trn:service_account:group/account-name)"`
}

// getServiceAccountOutput wraps the service account response.
type getServiceAccountOutput struct {
	ServiceAccount serviceAccount `json:"service_account" jsonschema:"The service account configuration"`
}

// GetServiceAccount returns an MCP tool for retrieving a service account.
func GetServiceAccount(tc *ToolContext) (mcp.Tool, mcp.ToolHandlerFor[getServiceAccountInput, getServiceAccountOutput]) {
	tool := mcp.Tool{
		Name:        "get_service_account",
		Description: "Retrieve a service account's configuration. Service accounts provide machine-to-machine authentication via OIDC trust policies or client credentials.",
		Annotations: &mcp.ToolAnnotations{
			Title:        "Get Service Account",
			ReadOnlyHint: true,
		},
	}

	handler := func(ctx context.Context, _ *mcp.CallToolRequest, input getServiceAccountInput) (*mcp.CallToolResult, getServiceAccountOutput, error) {
		fetchedModel, err := tc.servicesCatalog.FetchModel(ctx, input.ID)
		if err != nil {
			return nil, getServiceAccountOutput{}, WrapMCPToolError(err, "failed to resolve service account %q", input.ID)
		}

		sa, ok := fetchedModel.(*models.ServiceAccount)
		if !ok {
			return nil, getServiceAccountOutput{}, NewMCPToolError("service account with id %s not found", input.ID)
		}

		policies := make([]oidcTrustPolicy, len(sa.OIDCTrustPolicies))
		for i, p := range sa.OIDCTrustPolicies {
			policies[i] = oidcTrustPolicy{
				Issuer:          p.Issuer,
				BoundClaimsType: string(p.BoundClaimsType),
				BoundClaims:     p.BoundClaims,
			}
		}

		return nil, getServiceAccountOutput{
			ServiceAccount: serviceAccount{
				ServiceAccountID:         sa.GetGlobalID(),
				TRN:                      sa.Metadata.TRN,
				Name:                     sa.Name,
				Description:              sa.Description,
				GroupID:                  gid.ToGlobalID(types.GroupModelType, sa.GroupID),
				CreatedBy:                sa.CreatedBy,
				ClientCredentialsEnabled: sa.ClientCredentialsEnabled(),
				OIDCTrustPolicies:        policies,
			},
		}, nil
	}

	return tool, handler
}
