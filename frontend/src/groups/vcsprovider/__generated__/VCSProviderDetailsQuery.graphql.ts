/**
 * @generated SignedSource<<7ebc5a45234cf49af44c6449ec5731f7>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type VCSProviderType = "github" | "gitlab" | "%future added value";
export type VCSProviderDetailsQuery$variables = {
  id: string;
};
export type VCSProviderDetailsQuery$data = {
  readonly node: {
    readonly autoCreateWebhooks?: boolean;
    readonly createdBy?: string;
    readonly description?: string;
    readonly id?: string;
    readonly metadata?: {
      readonly createdAt: any;
      readonly trn: string;
    };
    readonly name?: string;
    readonly resourcePath?: string;
    readonly type?: VCSProviderType;
    readonly url?: string;
  } | null | undefined;
};
export type VCSProviderDetailsQuery = {
  response: VCSProviderDetailsQuery$data;
  variables: VCSProviderDetailsQuery$variables;
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
  "name": "name",
  "storageKey": null
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "createdBy",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "description",
  "storageKey": null
},
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "type",
  "storageKey": null
},
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "url",
  "storageKey": null
},
v8 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "resourcePath",
  "storageKey": null
},
v9 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "autoCreateWebhooks",
  "storageKey": null
},
v10 = {
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
    "name": "VCSProviderDetailsQuery",
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
              (v5/*: any*/),
              (v6/*: any*/),
              (v7/*: any*/),
              (v8/*: any*/),
              (v9/*: any*/),
              (v10/*: any*/)
            ],
            "type": "VCSProvider",
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
    "name": "VCSProviderDetailsQuery",
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
              (v5/*: any*/),
              (v6/*: any*/),
              (v7/*: any*/),
              (v8/*: any*/),
              (v9/*: any*/),
              (v10/*: any*/)
            ],
            "type": "VCSProvider",
            "abstractKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "6641d8c88edf47fff1ce53e490100ae0",
    "id": null,
    "metadata": {},
    "name": "VCSProviderDetailsQuery",
    "operationKind": "query",
    "text": "query VCSProviderDetailsQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ... on VCSProvider {\n      id\n      name\n      createdBy\n      description\n      type\n      url\n      resourcePath\n      autoCreateWebhooks\n      metadata {\n        createdAt\n        trn\n      }\n    }\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "7333b2d5e7da9030e72168f8b1db4167";

export default node;
