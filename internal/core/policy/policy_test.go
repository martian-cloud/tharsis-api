package policy

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

const aliasSourceID = "mi-source"

// managedIdentity builds an identity fixture with the TRN the db layer would have given it, since
// GetResourcePath parses the TRN and panics when it is missing.
func managedIdentity(id, groupPath, name string, aliasSourceID *string) models.ManagedIdentity {
	return models.ManagedIdentity{
		Name:          name,
		AliasSourceID: aliasSourceID,
		Metadata: models.ResourceMetadata{
			ID:  id,
			TRN: trn.TypeManagedIdentity.Build(groupPath, name),
		},
	}
}

// An alias in the workspace's own group pointing at an identity that lives elsewhere. Its own path
// names the alias; only the source lookup can produce the source's path.
func aliasIdentity() models.ManagedIdentity {
	sourceID := aliasSourceID
	return managedIdentity("mi-alias", "root/team", "aws-alias", &sourceID)
}

func sourceIdentity() models.ManagedIdentity {
	return managedIdentity(aliasSourceID, "other/prod", "aws", nil)
}

func plainIdentity() models.ManagedIdentity {
	return managedIdentity("mi-plain", "root/team", "aws", nil)
}

// expectSourceLookup expects the alias source lookup to be filtered to the alias's source rather than
// fetching an unbounded list.
func expectSourceLookup(mockManagedIdentities *db.MockManagedIdentities) {
	mockManagedIdentities.On("GetManagedIdentities", mock.Anything,
		mock.MatchedBy(func(input *db.GetManagedIdentitiesInput) bool {
			return input.Filter != nil &&
				assert.ObjectsAreEqual([]string{aliasSourceID}, input.Filter.ManagedIdentityIDs)
		}),
	).Return(&db.ManagedIdentitiesResult{
		ManagedIdentities: []models.ManagedIdentity{sourceIdentity()},
	}, nil)
}

func expectPolicies(mockPolicies *db.MockPolicies, policies ...*models.Policy) {
	mockPolicies.On("GetPolicies", mock.Anything, mock.Anything).
		Return(&db.PoliciesResult{Policies: policies}, nil)
}

func policyNames(policies []models.Policy) []string {
	var names []string
	for i := range policies {
		names = append(names, policies[i].Name)
	}
	return names
}

func TestGetWorkspaceAssignedPolicies(t *testing.T) {
	// Asserts the group lookup is filtered to exactly the workspace's ancestor groups, and returns one
	// group per path.
	expectGroups := func(mockGroups *db.MockGroups, paths ...string) {
		groups := make([]models.Group, 0, len(paths))
		for i, path := range paths {
			groups = append(groups, models.Group{
				FullPath: path,
				Metadata: models.ResourceMetadata{ID: fmt.Sprintf("g-%d", i)},
			})
		}
		mockGroups.On("GetGroups", mock.Anything,
			mock.MatchedBy(func(input *db.GetGroupsInput) bool {
				return input.Filter != nil && assert.ObjectsAreEqual(paths, input.Filter.GroupPaths)
			}),
		).Return(&db.GroupsResult{Groups: groups}, nil)
	}

	workspace := &models.Workspace{
		FullPath: "root/team/ws",
		Metadata: models.ResourceMetadata{ID: "ws-1"},
	}

	tests := []struct {
		name              string
		workspace         *models.Workspace
		managedIdentities []models.ManagedIdentity
		setupMocks        func(*db.MockGroups, *db.MockPolicies, *db.MockManagedIdentities)
		wantNames         []string
	}{
		{
			name:       "workspace with no ancestor groups returns no policies",
			workspace:  &models.Workspace{FullPath: "solo"},
			setupMocks: func(_ *db.MockGroups, _ *db.MockPolicies, _ *db.MockManagedIdentities) {},
			wantNames:  nil,
		},
		{
			name:      "no ancestor groups found returns no policies",
			workspace: workspace,
			setupMocks: func(mockGroups *db.MockGroups, _ *db.MockPolicies, _ *db.MockManagedIdentities) {
				mockGroups.On("GetGroups", mock.Anything, mock.Anything).
					Return(&db.GroupsResult{Groups: nil}, nil)
			},
			wantNames: nil,
		},
		{
			name:      "returns only policies whose scope matches the run",
			workspace: workspace,
			setupMocks: func(mockGroups *db.MockGroups, mockPolicies *db.MockPolicies, _ *db.MockManagedIdentities) {
				expectGroups(mockGroups, "root/team", "root")
				expectPolicies(mockPolicies,
					&models.Policy{Name: "applies-to-all"},
					&models.Policy{
						Name: "excludes-this-workspace",
						Scope: []*models.ScopeRule{
							{
								Type:    models.ScopeRuleTypeWorkspace,
								Action:  models.ScopeRuleActionExclude,
								Pattern: "root/team/ws",
							},
						},
					},
				)
			},
			wantNames: []string{"applies-to-all"},
		},
		{
			// The case that motivated alias resolution: the rule names where the identity actually
			// lives, which is nowhere in the alias's own path.
			name:              "managed identity rule matches an alias by its source path",
			workspace:         workspace,
			managedIdentities: []models.ManagedIdentity{aliasIdentity()},
			setupMocks: func(mockGroups *db.MockGroups, mockPolicies *db.MockPolicies, mockManagedIdentities *db.MockManagedIdentities) {
				expectGroups(mockGroups, "root/team", "root")
				expectSourceLookup(mockManagedIdentities)
				expectPolicies(mockPolicies, &models.Policy{
					Name: "matches-source-path",
					Scope: []*models.ScopeRule{
						{
							Type:    models.ScopeRuleTypeManagedIdentity,
							Action:  models.ScopeRuleActionInclude,
							Pattern: "other/prod/*",
						},
					},
				})
			},
			wantNames: []string{"matches-source-path"},
		},
		{
			// Resolving the source must add a path rather than replace the alias's own.
			name:              "managed identity rule still matches an alias by its own path",
			workspace:         workspace,
			managedIdentities: []models.ManagedIdentity{aliasIdentity()},
			setupMocks: func(mockGroups *db.MockGroups, mockPolicies *db.MockPolicies, mockManagedIdentities *db.MockManagedIdentities) {
				expectGroups(mockGroups, "root/team", "root")
				expectSourceLookup(mockManagedIdentities)
				expectPolicies(mockPolicies, &models.Policy{
					Name: "matches-alias-path",
					Scope: []*models.ScopeRule{
						{
							Type:    models.ScopeRuleTypeManagedIdentity,
							Action:  models.ScopeRuleActionInclude,
							Pattern: "root/team/aws-*",
						},
					},
				})
			},
			wantNames: []string{"matches-alias-path"},
		},
		{
			name:              "exclude by source path filters out a policy for an alias",
			workspace:         workspace,
			managedIdentities: []models.ManagedIdentity{aliasIdentity()},
			setupMocks: func(mockGroups *db.MockGroups, mockPolicies *db.MockPolicies, mockManagedIdentities *db.MockManagedIdentities) {
				expectGroups(mockGroups, "root/team", "root")
				expectSourceLookup(mockManagedIdentities)
				expectPolicies(mockPolicies, &models.Policy{
					Name: "excluded-by-source-path",
					Scope: []*models.ScopeRule{
						{
							Type:    models.ScopeRuleTypeManagedIdentity,
							Action:  models.ScopeRuleActionExclude,
							Pattern: "other/prod/aws",
						},
					},
				})
			},
			wantNames: nil,
		},
		{
			// No GetManagedIdentities expectation: an identity that is not an alias must not cost a
			// lookup, and the mock fails the test if one happens.
			name:              "no alias assigned means no source lookup",
			workspace:         workspace,
			managedIdentities: []models.ManagedIdentity{plainIdentity()},
			setupMocks: func(mockGroups *db.MockGroups, mockPolicies *db.MockPolicies, _ *db.MockManagedIdentities) {
				expectGroups(mockGroups, "root/team", "root")
				expectPolicies(mockPolicies, &models.Policy{
					Name: "matches-identity-path",
					Scope: []*models.ScopeRule{
						{
							Type:    models.ScopeRuleTypeManagedIdentity,
							Action:  models.ScopeRuleActionInclude,
							Pattern: "root/team/aws",
						},
					},
				})
			},
			wantNames: []string{"matches-identity-path"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGroups := db.NewMockGroups(t)
			mockPolicies := db.NewMockPolicies(t)
			mockManagedIdentities := db.NewMockManagedIdentities(t)
			tt.setupMocks(mockGroups, mockPolicies, mockManagedIdentities)

			dbClient := &db.Client{
				Groups:            mockGroups,
				Policies:          mockPolicies,
				ManagedIdentities: mockManagedIdentities,
			}

			got, err := GetWorkspaceAssignedPolicies(context.Background(), dbClient, tt.workspace, tt.managedIdentities, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.wantNames, policyNames(got))
		})
	}
}

func TestGetPoliciesReferencingManagedIdentity(t *testing.T) {
	tests := []struct {
		name            string
		managedIdentity models.ManagedIdentity
		setupMocks      func(*db.MockPolicies, *db.MockManagedIdentities)
		wantNames       []string
	}{
		{
			// Only a rule naming the identity counts. An unscoped policy fires on every run under its
			// group, this identity's included, but it singles out no identity — and a namespace rule
			// cannot be judged without knowing which workspace the run is in.
			name:            "returns only the policies naming the identity",
			managedIdentity: plainIdentity(),
			setupMocks: func(mockPolicies *db.MockPolicies, _ *db.MockManagedIdentities) {
				expectPolicies(mockPolicies,
					&models.Policy{
						Name: "names-the-identity",
						Scope: []*models.ScopeRule{
							{
								Type:    models.ScopeRuleTypeManagedIdentity,
								Action:  models.ScopeRuleActionInclude,
								Pattern: "root/team/aws",
							},
						},
					},
					&models.Policy{
						Name: "names-a-namespace",
						Scope: []*models.ScopeRule{
							{
								Type:    models.ScopeRuleTypeWorkspace,
								Action:  models.ScopeRuleActionInclude,
								Pattern: "root/team/*",
							},
						},
					},
					&models.Policy{
						Name: "names-another-identity",
						Scope: []*models.ScopeRule{
							{
								Type:    models.ScopeRuleTypeManagedIdentity,
								Action:  models.ScopeRuleActionInclude,
								Pattern: "root/team/azure",
							},
						},
					},
					&models.Policy{Name: "applies-to-all"},
				)
			},
			wantNames: []string{"names-the-identity"},
		},
		{
			// Excludes still win, so an identity named by both is not one the policy is assigned to.
			name:            "honours an exclude naming the identity",
			managedIdentity: plainIdentity(),
			setupMocks: func(mockPolicies *db.MockPolicies, _ *db.MockManagedIdentities) {
				expectPolicies(mockPolicies, &models.Policy{
					Name: "includes-then-excludes",
					Scope: []*models.ScopeRule{
						{
							Type:    models.ScopeRuleTypeManagedIdentity,
							Action:  models.ScopeRuleActionInclude,
							Pattern: "root/team/*",
						},
						{
							Type:    models.ScopeRuleTypeManagedIdentity,
							Action:  models.ScopeRuleActionExclude,
							Pattern: "root/team/aws",
						},
					},
				})
			},
			wantNames: nil,
		},
		{
			// An identity page has no workspace either, so alias resolution has to work here too.
			name:            "matches an alias by its source path",
			managedIdentity: aliasIdentity(),
			setupMocks: func(mockPolicies *db.MockPolicies, mockManagedIdentities *db.MockManagedIdentities) {
				expectSourceLookup(mockManagedIdentities)
				expectPolicies(mockPolicies, &models.Policy{
					Name: "matches-source-path",
					Scope: []*models.ScopeRule{
						{
							Type:    models.ScopeRuleTypeManagedIdentity,
							Action:  models.ScopeRuleActionInclude,
							Pattern: "other/prod/aws",
						},
					},
				})
			},
			wantNames: []string{"matches-source-path"},
		},
		{
			// The whole tree around the identity's group, in one query: a policy can be owned above it,
			// by it, or by a subgroup whose workspaces inherit the identity.
			name:            "queries the identity's whole group tree",
			managedIdentity: plainIdentity(),
			setupMocks: func(mockPolicies *db.MockPolicies, _ *db.MockManagedIdentities) {
				mockPolicies.On("GetPolicies", mock.Anything,
					mock.MatchedBy(func(input *db.GetPoliciesInput) bool {
						return input.Filter != nil &&
							input.Filter.GroupPathTree != nil &&
							*input.Filter.GroupPathTree == "root/team" &&
							input.Filter.GroupIDs == nil
					}),
				).Return(&db.PoliciesResult{
					Policies: []*models.Policy{
						{
							// Owned by a subgroup of the identity's group, which the ancestor-only
							// lookup this replaced could not reach.
							Name: "owned-by-a-nested-group",
							Scope: []*models.ScopeRule{
								{
									Type:    models.ScopeRuleTypeManagedIdentity,
									Action:  models.ScopeRuleActionInclude,
									Pattern: "root/team/aws",
								},
							},
						},
					},
				}, nil)
			},
			wantNames: []string{"owned-by-a-nested-group"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPolicies := db.NewMockPolicies(t)
			mockManagedIdentities := db.NewMockManagedIdentities(t)
			tt.setupMocks(mockPolicies, mockManagedIdentities)

			// No Groups mock: resolving an identity's policies takes a single policy query, so a group
			// lookup here would be a regression.
			dbClient := &db.Client{
				Policies:          mockPolicies,
				ManagedIdentities: mockManagedIdentities,
			}

			got, err := GetPoliciesReferencingManagedIdentity(context.Background(), dbClient, &tt.managedIdentity)
			require.NoError(t, err)
			assert.Equal(t, tt.wantNames, policyNames(got))
		})
	}
}
