/**
 * @generated SignedSource<<997579b0b0272dbfa901d4bb8a5c2a64>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type NamespaceFavoriteButtonQuery$variables = {
  namespacePath: string;
};
export type NamespaceFavoriteButtonQuery$data = {
  readonly me: {
    readonly namespaceFavorites?: {
      readonly edges: ReadonlyArray<{
        readonly node: {
          readonly id: string;
        };
      } | null | undefined> | null | undefined;
    };
  } | null | undefined;
};
export type NamespaceFavoriteButtonQuery = {
  response: NamespaceFavoriteButtonQuery$data;
  variables: NamespaceFavoriteButtonQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "namespacePath"
  }
],
v1 = [
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "id",
    "storageKey": null
  }
],
v2 = {
  "kind": "InlineFragment",
  "selections": [
    {
      "alias": null,
      "args": [
        {
          "kind": "Literal",
          "name": "first",
          "value": 1
        },
        {
          "kind": "Variable",
          "name": "namespacePath",
          "variableName": "namespacePath"
        }
      ],
      "concreteType": "NamespaceFavoriteConnection",
      "kind": "LinkedField",
      "name": "namespaceFavorites",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "concreteType": "NamespaceFavoriteEdge",
          "kind": "LinkedField",
          "name": "edges",
          "plural": true,
          "selections": [
            {
              "alias": null,
              "args": null,
              "concreteType": "NamespaceFavorite",
              "kind": "LinkedField",
              "name": "node",
              "plural": false,
              "selections": (v1/*: any*/),
              "storageKey": null
            }
          ],
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "User",
  "abstractKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "NamespaceFavoriteButtonQuery",
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": null,
        "kind": "LinkedField",
        "name": "me",
        "plural": false,
        "selections": [
          (v2/*: any*/)
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
    "name": "NamespaceFavoriteButtonQuery",
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": null,
        "kind": "LinkedField",
        "name": "me",
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
            "selections": (v1/*: any*/),
            "type": "Node",
            "abstractKey": "__isNode"
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "93306f941c5b0cc521a7553ad90b459b",
    "id": null,
    "metadata": {},
    "name": "NamespaceFavoriteButtonQuery",
    "operationKind": "query",
    "text": "query NamespaceFavoriteButtonQuery(\n  $namespacePath: String!\n) {\n  me {\n    __typename\n    ... on User {\n      namespaceFavorites(first: 1, namespacePath: $namespacePath) {\n        edges {\n          node {\n            id\n          }\n        }\n      }\n    }\n    ... on Node {\n      __isNode: __typename\n      id\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "1f59342abff526b0f4fb9886ee5f1fe3";

export default node;
