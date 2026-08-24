/**
 * @generated SignedSource<<5d6985355ef992dc6f94e329a50ed71b>>
 * @lightSyntaxTransform
 * @nogrep
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
  readonly " $fragmentSpreads": FragmentRefs<"AssignedManagedIdentityListFragment_assignedManagedIdentities" | "AssignedPolicyListFragment_workspace" | "ConfigurationVersionDetailsFragment_workspace" | "NamespaceActivityFragment_activity" | "NamespaceMembershipsFragment_memberships" | "ProviderMirrorsFragment_namespace" | "RunsFragment_runs" | "StateVersionsFragment_stateVersions" | "VariablesFragment_variables" | "WorkspaceDetailsIndexFragment_workspace" | "WorkspaceSettingsFragment_workspace">;
  readonly " $fragmentType": "WorkspaceDetailsFragment_workspace";
};
export type WorkspaceDetailsFragment_workspace$key = {
  readonly " $data"?: WorkspaceDetailsFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceDetailsFragment_workspace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "WorkspaceDetailsFragment_workspace",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "id",
      "storageKey": null
    },
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
      "name": "AssignedManagedIdentityListFragment_assignedManagedIdentities"
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
    }
  ],
  "type": "Workspace",
  "abstractKey": null
};

(node as any).hash = "408a5bd8627cb50884fe917b7ed68597";

export default node;
