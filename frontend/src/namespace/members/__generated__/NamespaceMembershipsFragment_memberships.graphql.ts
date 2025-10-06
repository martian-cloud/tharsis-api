/**
 * @generated SignedSource<<c4913c679098fae3d5a7ab705a9667f9>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type NamespaceMembershipsFragment_memberships$data = {
  readonly " $fragmentSpreads": FragmentRefs<"NamespaceMembershipsIndexFragment_memberships" | "NewNamespaceMembershipFragment_memberships">;
  readonly " $fragmentType": "NamespaceMembershipsFragment_memberships";
};
export type NamespaceMembershipsFragment_memberships$key = {
  readonly " $data"?: NamespaceMembershipsFragment_memberships$data;
  readonly " $fragmentSpreads": FragmentRefs<"NamespaceMembershipsFragment_memberships">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "NamespaceMembershipsFragment_memberships",
  "selections": [
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "NamespaceMembershipsIndexFragment_memberships"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "NewNamespaceMembershipFragment_memberships"
    }
  ],
  "type": "Namespace",
  "abstractKey": "__isNamespace"
};

(node as any).hash = "7ffe3173d1f476912949bc01ba51c94b";

export default node;
