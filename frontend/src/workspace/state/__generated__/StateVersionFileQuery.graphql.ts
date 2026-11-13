/**
 * @generated SignedSource<<896f9ae3839765bd6fdf0f6ee5efcb0a>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type StateVersionFileQuery$variables = {
  id: string;
};
export type StateVersionFileQuery$data = {
  readonly node: {
    readonly data?: string;
  } | null | undefined;
};
export type StateVersionFileQuery = {
  response: StateVersionFileQuery$data;
  variables: StateVersionFileQuery$variables;
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
  "kind": "InlineFragment",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "data",
      "storageKey": null
    }
  ],
  "type": "StateVersion",
  "abstractKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*:: as any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "StateVersionFileQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*:: as any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          (v2/*:: as any*/)
        ],
        "storageKey": null
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*:: as any*/),
    "kind": "Operation",
    "name": "StateVersionFileQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*:: as any*/),
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
          (v2/*:: as any*/),
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "id",
            "storageKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "f82b5c82e4f25935be7dcd011874fd4d",
    "id": null,
    "metadata": {},
    "name": "StateVersionFileQuery",
    "operationKind": "query",
    "text": "query StateVersionFileQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ... on StateVersion {\n      data\n    }\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "fbf017c8be87ca4a4e9e9243ddc8fb1d";

export default node;
