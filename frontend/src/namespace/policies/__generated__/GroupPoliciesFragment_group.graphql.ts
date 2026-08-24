/**
 * @generated SignedSource<<672688841ea386970a9479f0db0c818e>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupPoliciesFragment_group$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "GroupPoliciesFragment_group";
};
export type GroupPoliciesFragment_group$key = {
  readonly " $data"?: GroupPoliciesFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"GroupPoliciesFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "GroupPoliciesFragment_group",
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
    }
  ],
  "type": "Group",
  "abstractKey": null
};

(node as any).hash = "5d09c3d10de4ab7e671eae55773f605c";

export default node;
