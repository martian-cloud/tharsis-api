/**
 * @generated SignedSource<<41fbe58108bf9980f99918e111e38746>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type StateVersionDetailsQuery$variables = {
  id: string;
};
export type StateVersionDetailsQuery$data = {
  readonly node: {
    readonly createdBy?: string;
    readonly id?: string;
    readonly metadata?: {
      readonly createdAt: any;
      readonly trn: string;
    };
    readonly run?: {
      readonly createdBy: string;
    } | null | undefined;
    readonly " $fragmentSpreads": FragmentRefs<"StateVersionFileFragment_stateVersion">;
  } | null | undefined;
};
export type StateVersionDetailsQuery = {
  response: StateVersionDetailsQuery$data;
  variables: StateVersionDetailsQuery$variables;
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
  "name": "createdBy",
  "storageKey": null
},
v4 = {
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
      "name": "createdAt",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "trn",
      "storageKey": null
    }
  ],
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "StateVersionDetailsQuery",
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
              (v4/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "Run",
                "kind": "LinkedField",
                "name": "run",
                "plural": false,
                "selections": [
                  (v3/*: any*/)
                ],
                "storageKey": null
              },
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "StateVersionFileFragment_stateVersion"
              }
            ],
            "type": "StateVersion",
            "abstractKey": null
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
    "name": "StateVersionDetailsQuery",
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
              (v4/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "Run",
                "kind": "LinkedField",
                "name": "run",
                "plural": false,
                "selections": [
                  (v3/*: any*/),
                  (v2/*: any*/)
                ],
                "storageKey": null
              }
            ],
            "type": "StateVersion",
            "abstractKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "c00336a37d4cd2edeedcd06b2c24029b",
    "id": null,
    "metadata": {},
    "name": "StateVersionDetailsQuery",
    "operationKind": "query",
    "text": "query StateVersionDetailsQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ... on StateVersion {\n      id\n      createdBy\n      metadata {\n        createdAt\n        trn\n      }\n      run {\n        createdBy\n        id\n      }\n      ...StateVersionFileFragment_stateVersion\n    }\n    id\n  }\n}\n\nfragment StateVersionFileFragment_stateVersion on StateVersion {\n  id\n}\n"
  }
};
})();

(node as any).hash = "3af09e64ae50bd1228eeaafc40116f07";

export default node;
