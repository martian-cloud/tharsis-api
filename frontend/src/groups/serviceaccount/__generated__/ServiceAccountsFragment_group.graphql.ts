/**
 * @generated SignedSource<<812074511aa9859525df32b0b50ca841>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ServiceAccountsFragment_group$data = {
  readonly " $fragmentSpreads": FragmentRefs<"EditServiceAccountFragment_group" | "NewServiceAccountFragment_group" | "ServiceAccountDetailsFragment_group" | "ServiceAccountListFragment_group">;
  readonly " $fragmentType": "ServiceAccountsFragment_group";
};
export type ServiceAccountsFragment_group$key = {
  readonly " $data"?: ServiceAccountsFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"ServiceAccountsFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ServiceAccountsFragment_group",
  "selections": [
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "ServiceAccountListFragment_group"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "ServiceAccountDetailsFragment_group"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "NewServiceAccountFragment_group"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "EditServiceAccountFragment_group"
    }
  ],
  "type": "Group",
  "abstractKey": null
};

(node as any).hash = "7cac47900478d6ed35dfb1ef02294649";

export default node;
