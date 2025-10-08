/**
 * @generated SignedSource<<6a75e20c6cc2237ba981f5248b62e3dc>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ManagedIdentitiesFragment_group$data = {
  readonly " $fragmentSpreads": FragmentRefs<"EditManagedIdentityFragment_group" | "ManagedIdentityDetailsFragment_group" | "ManagedIdentityListFragment_group" | "NewManagedIdentityFragment_group">;
  readonly " $fragmentType": "ManagedIdentitiesFragment_group";
};
export type ManagedIdentitiesFragment_group$key = {
  readonly " $data"?: ManagedIdentitiesFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"ManagedIdentitiesFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ManagedIdentitiesFragment_group",
  "selections": [
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "ManagedIdentityListFragment_group"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "NewManagedIdentityFragment_group"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "EditManagedIdentityFragment_group"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "ManagedIdentityDetailsFragment_group"
    }
  ],
  "type": "Group",
  "abstractKey": null
};

(node as any).hash = "a7a7f0e392d3f0552a5cc49c4b3233a3";

export default node;
