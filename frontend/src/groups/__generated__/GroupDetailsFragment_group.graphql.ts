/**
 * @generated SignedSource<<3af89d3df86076761dde44a304469f8d>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupDetailsFragment_group$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly name: string;
  readonly " $fragmentSpreads": FragmentRefs<"CleanupPoliciesFragment_namespace" | "FederatedRegistriesFragment_group" | "GPGKeysFragment_group" | "GroupDetailsIndexFragment_group" | "GroupPackagesFragment_group" | "GroupPoliciesFragment_group" | "GroupRunnersFragment_group" | "GroupRunsFragment_group" | "GroupSettingsFragment_group" | "ManagedIdentitiesFragment_group" | "NamespaceActivityFragment_activity" | "NamespaceMembershipsFragment_memberships" | "ProviderMirrorsFragment_namespace" | "ServiceAccountsFragment_group" | "TerraformModulesFragment_group" | "VCSProvidersFragment_group" | "VariablesFragment_variables">;
  readonly " $fragmentType": "GroupDetailsFragment_group";
};
export type GroupDetailsFragment_group$key = {
  readonly " $data"?: GroupDetailsFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"GroupDetailsFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "GroupDetailsFragment_group",
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
      "name": "fullPath",
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
      "args": null,
      "kind": "FragmentSpread",
      "name": "GroupDetailsIndexFragment_group"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "ManagedIdentitiesFragment_group"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "GroupRunnersFragment_group"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "GroupRunsFragment_group"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "ServiceAccountsFragment_group"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "TerraformModulesFragment_group"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "GroupPackagesFragment_group"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "GroupPoliciesFragment_group"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "VCSProvidersFragment_group"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "FederatedRegistriesFragment_group"
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
      "name": "GPGKeysFragment_group"
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
      "name": "NamespaceActivityFragment_activity"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "GroupSettingsFragment_group"
    }
  ],
  "type": "Group",
  "abstractKey": null
};

(node as any).hash = "08e0cbf75e049a51daeaad7dfb6f1aac";

export default node;
