/**
 * @generated SignedSource<<74faef2db3c2c7cdb867f831c9219663>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type FederatedRegistriesFragment_group$data = {
  readonly " $fragmentSpreads": FragmentRefs<"EditFederatedRegistryFragment_group" | "FederatedRegistryDetailsFragment_group" | "FederatedRegistryListFragment_group" | "NewFederatedRegistryFragment_group">;
  readonly " $fragmentType": "FederatedRegistriesFragment_group";
};
export type FederatedRegistriesFragment_group$key = {
  readonly " $data"?: FederatedRegistriesFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"FederatedRegistriesFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "FederatedRegistriesFragment_group",
  "selections": [
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "FederatedRegistryListFragment_group"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "FederatedRegistryDetailsFragment_group"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "NewFederatedRegistryFragment_group"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "EditFederatedRegistryFragment_group"
    }
  ],
  "type": "Group",
  "abstractKey": null
};

(node as any).hash = "195b0365e9c3389c2518358f48a74944";

export default node;
