/**
 * @generated SignedSource<<e2f3b1379215a108de88648a1e43a859>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type CleanupPoliciesFragment_namespace$data = {
  readonly fullPath: string;
  readonly " $fragmentType": "CleanupPoliciesFragment_namespace";
};
export type CleanupPoliciesFragment_namespace$key = {
  readonly " $data"?: CleanupPoliciesFragment_namespace$data;
  readonly " $fragmentSpreads": FragmentRefs<"CleanupPoliciesFragment_namespace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "CleanupPoliciesFragment_namespace",
  "selections": [
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

(node as any).hash = "cb64f00c13bffea9d3c40dff04cad9a6";

export default node;
