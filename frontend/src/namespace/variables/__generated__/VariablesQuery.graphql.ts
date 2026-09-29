/**
 * @generated SignedSource<<857204be3706bafb2b313eb3303f91c4>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type VariableCategory = "environment" | "terraform" | "%future added value";
export type VariablesQuery$variables = {
  id: string;
};
export type VariablesQuery$data = {
  readonly node: {
    readonly fullPath?: string;
    readonly id?: string;
    readonly variables?: ReadonlyArray<{
      readonly category: VariableCategory;
      readonly id: string;
      readonly key: string;
      readonly " $fragmentSpreads": FragmentRefs<"VariableListItemFragment_variable">;
    }>;
  } | null | undefined;
};
export type VariablesQuery = {
  response: VariablesQuery$data;
  variables: VariablesQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "id"
  }
],
v1 = [
  {
    "kind": "Variable",
    "name": "id",
    "variableName": "id"
  }
],
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "fullPath",
  "storageKey": null
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "key",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "category",
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "VariablesQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          {
            "kind": "InlineFragment",
            "selections": [
              (v2/*: any*/),
              (v3/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "NamespaceVariable",
                "kind": "LinkedField",
                "name": "variables",
                "plural": true,
                "selections": [
                  (v2/*: any*/),
                  (v4/*: any*/),
                  (v5/*: any*/),
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
          }
        ],
        "storageKey": null
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "VariablesQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "__typename",
            "storageKey": null
          },
          (v2/*: any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              (v3/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "NamespaceVariable",
                "kind": "LinkedField",
                "name": "variables",
                "plural": true,
                "selections": [
                  (v2/*: any*/),
                  (v4/*: any*/),
                  (v5/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "sensitive",
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
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "latestVersionId",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "ResourceMetadata",
                    "kind": "LinkedField",
                    "name": "metadata",
                    "plural": false,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "updatedAt",
                        "storageKey": null
                      }
                    ],
                    "storageKey": null
                  }
                ],
                "storageKey": null
              }
            ],
            "type": "Namespace",
            "abstractKey": "__isNamespace"
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "88c6e59b60a326174920a6ecc15e5d28",
    "id": null,
    "metadata": {},
    "name": "VariablesQuery",
    "operationKind": "query",
    "text": "query VariablesQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ... on Namespace {\n      __isNamespace: __typename\n      id\n      fullPath\n      variables {\n        id\n        key\n        category\n        ...VariableListItemFragment_variable\n      }\n    }\n    id\n  }\n}\n\nfragment VariableListItemFragment_variable on NamespaceVariable {\n  id\n  key\n  category\n  sensitive\n  value\n  namespacePath\n  latestVersionId\n  metadata {\n    updatedAt\n  }\n}\n"
  }
};
})();

(node as any).hash = "b37d2fc2e5ce2ce2b54f409e2a984ce7";

export default node;
