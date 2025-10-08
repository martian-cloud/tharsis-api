/**
 * @generated SignedSource<<3017ff9a2084bb699fb27a6016438c3e>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type VariableCategory = "environment" | "terraform" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type InheritedVariableListItemFragment_variable$data = {
  readonly category: VariableCategory;
  readonly id: string;
  readonly key: string;
  readonly namespacePath: string;
  readonly value: string | null | undefined;
  readonly " $fragmentType": "InheritedVariableListItemFragment_variable";
};
export type InheritedVariableListItemFragment_variable$key = {
  readonly " $data"?: InheritedVariableListItemFragment_variable$data;
  readonly " $fragmentSpreads": FragmentRefs<"InheritedVariableListItemFragment_variable">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "InheritedVariableListItemFragment_variable",
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
      "name": "key",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "category",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "value",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "namespacePath",
      "storageKey": null
    }
  ],
  "type": "NamespaceVariable",
  "abstractKey": null
};

(node as any).hash = "3dd47fb935dd69b7bbe3098d277ff61c";

export default node;
