/**
 * @generated SignedSource<<ca7b9de5a3f5886b4dd359fdf8d1be5b>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type NamespaceMembershipListFragment_memberships$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "NamespaceMembershipListFragment_memberships";
};
export type NamespaceMembershipListFragment_memberships$key = {
  readonly " $data"?: NamespaceMembershipListFragment_memberships$data;
  readonly " $fragmentSpreads": FragmentRefs<"NamespaceMembershipListFragment_memberships">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "NamespaceMembershipListFragment_memberships",
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
  "type": "Namespace",
  "abstractKey": "__isNamespace"
};

(node as any).hash = "c2b2b08447895b0b67c52982e23eac88";

export default node;
