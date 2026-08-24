// Package policy provides shared core logic for policy evaluation.
package policy

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	nsutils "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace/utils"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// GetWorkspaceAssignedPolicies returns the policies that apply to a run in the given workspace,
// filtered by optional stage. managedIdentities are the identities the workspace uses, which is what
// the managed identity scope rules are matched against. It performs no authorization — callers are
// responsible for permission checks.
//
// The candidate policies come from the workspace's ancestor groups, since a policy applies downward
// from the group that owns it.
func GetWorkspaceAssignedPolicies(
	ctx context.Context,
	dbClient *db.Client,
	ws *models.Workspace,
	managedIdentities []models.ManagedIdentity,
	stage *models.RunTaskStageName,
) ([]models.Policy, error) {
	// ExpandPath returns paths from the full path down to the root, so the workspace path is the first
	// element; drop it (no group exists at a workspace path, so it would silently produce no results).
	ancestorGroupPaths := nsutils.ExpandPath(ws.FullPath)[1:]
	if len(ancestorGroupPaths) == 0 {
		return nil, nil
	}

	groupsResult, err := dbClient.Groups.GetGroups(ctx, &db.GetGroupsInput{
		Filter: &db.GroupFilter{GroupPaths: ancestorGroupPaths},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get ancestor groups")
	}

	groupIDs := make([]string, 0, len(groupsResult.Groups))
	for i := range groupsResult.Groups {
		groupIDs = append(groupIDs, groupsResult.Groups[i].Metadata.ID)
	}
	if len(groupIDs) == 0 {
		return nil, nil
	}

	miPaths, err := managedIdentityPaths(ctx, dbClient, managedIdentities)
	if err != nil {
		return nil, err
	}

	policiesResult, err := dbClient.Policies.GetPolicies(ctx, &db.GetPoliciesInput{
		Filter: &db.PolicyFilter{
			GroupIDs: groupIDs,
			Stage:    stage,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get policies")
	}

	var assigned []models.Policy
	for _, p := range policiesResult.Policies {
		if p.MatchesRun(ws.FullPath, miPaths) {
			assigned = append(assigned, *p)
		}
	}
	return assigned, nil
}

// GetPoliciesReferencingManagedIdentity returns every policy that names the given managed identity in a
// managed identity scope rule. It performs no authorization — callers are responsible for permission
// checks.
//
// The candidates are the policies of the identity's own group, its ancestors, and everything nested
// beneath it. Those are exactly the groups whose policies can reach a run that uses the identity: a
// workspace that can use it lives in the identity's group subtree, and the policies applying to such a
// run are owned by that workspace's ancestors — which is either a group above the identity's or one
// between it and the workspace.
//
// A policy with no scope, or one scoped to a namespace, is not returned even though it may well fire on
// a run using this identity — that depends on which workspace the run is in. See
// models.Policy.MatchesManagedIdentity.
func GetPoliciesReferencingManagedIdentity(
	ctx context.Context,
	dbClient *db.Client,
	managedIdentity *models.ManagedIdentity,
) ([]models.Policy, error) {
	miPaths, err := managedIdentityPaths(ctx, dbClient, []models.ManagedIdentity{*managedIdentity})
	if err != nil {
		return nil, err
	}

	groupPath := managedIdentity.GetGroupPath()
	policiesResult, err := dbClient.Policies.GetPolicies(ctx, &db.GetPoliciesInput{
		Filter: &db.PolicyFilter{GroupPathTree: &groupPath},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get policies")
	}

	var referencing []models.Policy
	for _, p := range policiesResult.Policies {
		if p.MatchesManagedIdentity(miPaths) {
			referencing = append(referencing, *p)
		}
	}
	return referencing, nil
}

// managedIdentityPaths returns the paths the managed identity scope rules are matched against: each
// identity's own path, plus the path of the identity an alias points at. An alias lives in its own
// group under its own name, so its path says nothing about the identity it aliases, and a rule naming
// the source has to match too.
func managedIdentityPaths(
	ctx context.Context,
	dbClient *db.Client,
	managedIdentities []models.ManagedIdentity,
) ([]string, error) {
	miPaths := make([]string, 0, len(managedIdentities))
	var aliasSourceIDs []string
	for i := range managedIdentities {
		miPaths = append(miPaths, managedIdentities[i].GetResourcePath())
		if managedIdentities[i].AliasSourceID != nil {
			aliasSourceIDs = append(aliasSourceIDs, *managedIdentities[i].AliasSourceID)
		}
	}

	if len(aliasSourceIDs) == 0 {
		return miPaths, nil
	}

	// An alias cannot itself be aliased, so one lookup resolves every source.
	sourcesResult, err := dbClient.ManagedIdentities.GetManagedIdentities(ctx, &db.GetManagedIdentitiesInput{
		Filter: &db.ManagedIdentityFilter{ManagedIdentityIDs: aliasSourceIDs},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get alias source managed identities")
	}
	for i := range sourcesResult.ManagedIdentities {
		miPaths = append(miPaths, sourcesResult.ManagedIdentities[i].GetResourcePath())
	}

	return miPaths, nil
}
