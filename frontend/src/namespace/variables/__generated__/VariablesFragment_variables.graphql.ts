/**
 * @generated SignedSource<<3bc40d8fa7e363c728e57d65301cd156>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type VariablesFragment_variables$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "VariablesFragment_variables";
};
export type VariablesFragment_variables$key = {
  readonly " $data"?: VariablesFragment_variables$data;
  readonly " $fragmentSpreads": FragmentRefs<"VariablesFragment_variables">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "VariablesFragment_variables",
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

(node as any).hash = "e27ceff77104cb513ea9285f870c94a3";

export default node;
