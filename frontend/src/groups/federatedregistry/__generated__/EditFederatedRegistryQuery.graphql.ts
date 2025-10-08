/**
 * @generated SignedSource<<f7f15cf7d26caab5779b6f694916d56f>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type EditFederatedRegistryQuery$variables = {
  id: string;
};
export type EditFederatedRegistryQuery$data = {
  readonly node: {
    readonly audience?: string;
    readonly hostname?: string;
  } | null | undefined;
};
export type EditFederatedRegistryQuery = {
  response: EditFederatedRegistryQuery$data;
  variables: EditFederatedRegistryQuery$variables;
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
      "name": "hostname",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "audience",
      "storageKey": null
    }
  ],
  "type": "FederatedRegistry",
  "abstractKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "EditFederatedRegistryQuery",
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
    "name": "EditFederatedRegistryQuery",
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
    "cacheID": "09953aff00f642ade7c161e7f65220fb",
    "id": null,
    "metadata": {},
    "name": "EditFederatedRegistryQuery",
    "operationKind": "query",
    "text": "query EditFederatedRegistryQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ... on FederatedRegistry {\n      hostname\n      audience\n    }\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "6b46788bedfa3559d07242375f674b18";

export default node;
