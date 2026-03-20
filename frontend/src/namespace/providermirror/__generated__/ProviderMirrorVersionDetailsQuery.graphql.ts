/**
 * @generated SignedSource<<201ea03291f7472eec290efbf405ce9f>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProviderMirrorVersionDetailsQuery$variables = {
  id: string;
};
export type ProviderMirrorVersionDetailsQuery$data = {
  readonly node: {
    readonly createdBy?: string;
    readonly groupPath?: string;
    readonly id?: string;
    readonly metadata?: {
      readonly createdAt: any;
      readonly trn: string;
    };
    readonly platformMirrors?: ReadonlyArray<{
      readonly arch: string;
      readonly id: string;
      readonly os: string;
    }>;
    readonly providerAddress?: string;
    readonly version?: string;
  } | null | undefined;
};
export type ProviderMirrorVersionDetailsQuery = {
  response: ProviderMirrorVersionDetailsQuery$data;
  variables: ProviderMirrorVersionDetailsQuery$variables;
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
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "version",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "createdBy",
  "storageKey": null
},
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "providerAddress",
  "storageKey": null
},
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "groupPath",
  "storageKey": null
},
v8 = {
  "alias": null,
  "args": null,
  "concreteType": "TerraformProviderPlatformMirror",
  "kind": "LinkedField",
  "name": "platformMirrors",
  "plural": true,
  "selections": [
    (v2/*: any*/),
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "os",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "arch",
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
    "name": "ProviderMirrorVersionDetailsQuery",
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
              (v8/*: any*/)
            ],
            "type": "TerraformProviderVersionMirror",
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
    "name": "ProviderMirrorVersionDetailsQuery",
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
              (v8/*: any*/)
            ],
            "type": "TerraformProviderVersionMirror",
            "abstractKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "bd87d8de1b5fb4ac457a6811b768b54f",
    "id": null,
    "metadata": {},
    "name": "ProviderMirrorVersionDetailsQuery",
    "operationKind": "query",
    "text": "query ProviderMirrorVersionDetailsQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ... on TerraformProviderVersionMirror {\n      id\n      metadata {\n        createdAt\n        trn\n      }\n      version\n      createdBy\n      providerAddress\n      groupPath\n      platformMirrors {\n        id\n        os\n        arch\n      }\n    }\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "ee4ea07946021cd3fdbae0006bda785f";

export default node;
