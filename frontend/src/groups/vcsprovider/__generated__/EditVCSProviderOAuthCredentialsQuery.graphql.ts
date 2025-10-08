/**
 * @generated SignedSource<<680ac286976a4867a96dba4c003c1ac0>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type VCSProviderType = "github" | "gitlab" | "%future added value";
export type EditVCSProviderOAuthCredentialsQuery$variables = {
  id: string;
};
export type EditVCSProviderOAuthCredentialsQuery$data = {
  readonly node: {
    readonly name?: string;
    readonly type?: VCSProviderType;
  } | null | undefined;
};
export type EditVCSProviderOAuthCredentialsQuery = {
  response: EditVCSProviderOAuthCredentialsQuery$data;
  variables: EditVCSProviderOAuthCredentialsQuery$variables;
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
      "name": "name",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "type",
      "storageKey": null
    }
  ],
  "type": "VCSProvider",
  "abstractKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "EditVCSProviderOAuthCredentialsQuery",
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
    "name": "EditVCSProviderOAuthCredentialsQuery",
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
    "cacheID": "c0559abb9f7404a338b7d2291a855532",
    "id": null,
    "metadata": {},
    "name": "EditVCSProviderOAuthCredentialsQuery",
    "operationKind": "query",
    "text": "query EditVCSProviderOAuthCredentialsQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ... on VCSProvider {\n      name\n      type\n    }\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "d1b4a7cb4c3d2b08efbe7eb6cb0aa9de";

export default node;
