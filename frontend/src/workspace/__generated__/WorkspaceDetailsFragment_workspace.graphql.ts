/**
 * @generated SignedSource<<5d71d91c0391f503c3c6fd02922d43d7>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type WorkspaceDetailsFragment_workspace$data = {
  readonly description: string;
  readonly fullPath: string;
  readonly id: string;
  readonly name: string;
  readonly workspaceVcsProviderLink: {
    readonly id: string;
  } | null | undefined;
  readonly " $fragmentSpreads": FragmentRefs<"AssignedManagedIdentityListFragment_workspace" | "AssignedPolicyListFragment_workspace" | "CleanupPoliciesFragment_namespace" | "ConfigurationVersionDetailsFragment_workspace" | "NamespaceActivityFragment_activity" | "NamespaceMembershipsFragment_memberships" | "ProviderMirrorsFragment_namespace" | "RunsFragment_runs" | "StateVersionsFragment_stateVersions" | "VCSEventsFragment_vcsEvents" | "VariablesFragment_variables" | "WorkspaceDetailsIndexFragment_workspace" | "WorkspaceRoleBindingFragment_workspace" | "WorkspaceSettingsFragment_workspace">;
  readonly " $fragmentType": "WorkspaceDetailsFragment_workspace";
};
export type WorkspaceDetailsFragment_workspace$key = {
  readonly " $data"?: WorkspaceDetailsFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceDetailsFragment_workspace">;
};

const node: ReaderFragment = (function(){
var v0 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
};
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "WorkspaceDetailsFragment_workspace",
  "selections": [
    (v0/*:: as any*/),
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "name",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "description",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "fullPath",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "WorkspaceVCSProviderLink",
      "kind": "LinkedField",
      "name": "workspaceVcsProviderLink",
      "plural": false,
      "selections": [
        (v0/*:: as any*/)
      ],
      "storageKey": null
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "WorkspaceDetailsIndexFragment_workspace"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "AssignedPolicyListFragment_workspace"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "AssignedManagedIdentityListFragment_workspace"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "RunsFragment_runs"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "ConfigurationVersionDetailsFragment_workspace"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "StateVersionsFragment_stateVersions"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "VCSEventsFragment_vcsEvents"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "VariablesFragment_variables"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "NamespaceMembershipsFragment_memberships"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "WorkspaceSettingsFragment_workspace"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "NamespaceActivityFragment_activity"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "ProviderMirrorsFragment_namespace"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "CleanupPoliciesFragment_namespace"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "WorkspaceRoleBindingFragment_workspace"
    }
  ],
  "type": "Workspace",
  "abstractKey": null
};
})();

(node as any).hash = "647e283335154b8da266514b5ee1719a";

export default node;
