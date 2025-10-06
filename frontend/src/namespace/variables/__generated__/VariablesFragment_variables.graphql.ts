/**
 * @generated SignedSource<<648deda43b2f28540f99dfdc0c9144f9>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type VariableCategory = "environment" | "terraform" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type VariablesFragment_variables$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly variables: ReadonlyArray<{
    readonly category: VariableCategory;
    readonly id: string;
    readonly key: string;
    readonly " $fragmentSpreads": FragmentRefs<"VariableListItemFragment_variable">;
  }>;
  readonly " $fragmentType": "VariablesFragment_variables";
};
export type VariablesFragment_variables$key = {
  readonly " $data"?: VariablesFragment_variables$data;
  readonly " $fragmentSpreads": FragmentRefs<"VariablesFragment_variables">;
};

const node: ReaderFragment = (function(){
var v0 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
};
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "VariablesFragment_variables",
  "selections": [
    (v0/*: any*/),
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
      "concreteType": "NamespaceVariable",
      "kind": "LinkedField",
      "name": "variables",
      "plural": true,
      "selections": [
        (v0/*: any*/),
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
          "args": null,
          "kind": "FragmentSpread",
          "name": "VariableListItemFragment_variable"
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Namespace",
  "abstractKey": "__isNamespace"
};
})();

(node as any).hash = "68e4a22ff18e0d30ddd46cb27f748a88";

export default node;
