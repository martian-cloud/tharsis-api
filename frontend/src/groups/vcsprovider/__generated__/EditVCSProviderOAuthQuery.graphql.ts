/**
 * @generated SignedSource<<f260dd43b6c12f0db091ee0a51a885fd>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest, Query } from 'relay-runtime';
export type VCSProviderType = "github" | "gitlab" | "%future added value";
export type EditVCSProviderOAuthQuery$variables = {
  id: string;
};
export type EditVCSProviderOAuthQuery$data = {
  readonly node: {
    readonly name?: string;
    readonly type?: VCSProviderType;
  } | null;
};
export type EditVCSProviderOAuthQuery = {
  response: EditVCSProviderOAuthQuery$data;
  variables: EditVCSProviderOAuthQuery$variables;
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
    "name": "EditVCSProviderOAuthQuery",
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
    "name": "EditVCSProviderOAuthQuery",
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
    "cacheID": "450ee676d9a5cb05349ce383ec5d49d6",
    "id": null,
    "metadata": {},
    "name": "EditVCSProviderOAuthQuery",
    "operationKind": "query",
    "text": "query EditVCSProviderOAuthQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ... on VCSProvider {\n      name\n      type\n    }\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "632627788a8ce5b8e750f5df801d4e4a";

export default node;
