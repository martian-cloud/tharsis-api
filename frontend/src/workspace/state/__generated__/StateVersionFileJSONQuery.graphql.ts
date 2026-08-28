/**
 * @generated SignedSource<<7ec250f301712c8c9c8d8fd7815b4824>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type StateVersionFileJSONQuery$variables = {
  id: string;
};
export type StateVersionFileJSONQuery$data = {
  readonly node: {
    readonly jsonData?: string | null | undefined;
  } | null | undefined;
};
export type StateVersionFileJSONQuery = {
  response: StateVersionFileJSONQuery$data;
  variables: StateVersionFileJSONQuery$variables;
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
      "name": "jsonData",
      "storageKey": null
    }
  ],
  "type": "StateVersion",
  "abstractKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "StateVersionFileJSONQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
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
    "name": "StateVersionFileJSONQuery",
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
    "cacheID": "6034fc6a62652207f86bd379407068ef",
    "id": null,
    "metadata": {},
    "name": "StateVersionFileJSONQuery",
    "operationKind": "query",
    "text": "query StateVersionFileJSONQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ... on StateVersion {\n      jsonData\n    }\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "2f379d8b89ed98e645a4699169afbc2e";

export default node;
