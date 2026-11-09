/**
 * @generated SignedSource<<223d417f0e0c134d77bdbf69d6900127>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type TerraformProviderDocItemCategory = "DATA_SOURCES" | "EPHEMERAL_RESOURCES" | "FUNCTIONS" | "GUIDES" | "OVERVIEW" | "RESOURCES" | "%future added value";
export type TerraformProviderVersionDocFileQuery$variables = {
  category: TerraformProviderDocItemCategory;
  filename: string;
  id: string;
};
export type TerraformProviderVersionDocFileQuery$data = {
  readonly node: {
    readonly docs?: {
      readonly fileData: string | null | undefined;
    };
  } | null | undefined;
};
export type TerraformProviderVersionDocFileQuery = {
  response: TerraformProviderVersionDocFileQuery$data;
  variables: TerraformProviderVersionDocFileQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "category"
},
v1 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "filename"
},
v2 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "id"
},
v3 = [
  {
    "kind": "Variable",
    "name": "id",
    "variableName": "id"
  }
],
v4 = {
  "kind": "InlineFragment",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "TerraformProviderVersionDocs",
      "kind": "LinkedField",
      "name": "docs",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": [
            {
              "kind": "Variable",
              "name": "category",
              "variableName": "category"
            },
            {
              "kind": "Variable",
              "name": "filename",
              "variableName": "filename"
            }
          ],
          "kind": "ScalarField",
          "name": "fileData",
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "TerraformProviderVersion",
  "abstractKey": null
};
return {
  "fragment": {
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/),
      (v2/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "TerraformProviderVersionDocFileQuery",
    "selections": [
      {
        "alias": null,
        "args": (v3/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          (v4/*: any*/)
        ],
        "storageKey": null
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [
      (v2/*: any*/),
      (v0/*: any*/),
      (v1/*: any*/)
    ],
    "kind": "Operation",
    "name": "TerraformProviderVersionDocFileQuery",
    "selections": [
      {
        "alias": null,
        "args": (v3/*: any*/),
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
          (v4/*: any*/),
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
    "cacheID": "bf10738f20221b01ec5351a6c18bc1b6",
    "id": null,
    "metadata": {},
    "name": "TerraformProviderVersionDocFileQuery",
    "operationKind": "query",
    "text": "query TerraformProviderVersionDocFileQuery(\n  $id: String!\n  $category: TerraformProviderDocItemCategory!\n  $filename: String!\n) {\n  node(id: $id) {\n    __typename\n    ... on TerraformProviderVersion {\n      docs {\n        fileData(category: $category, filename: $filename)\n      }\n    }\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "377f353b84efe9d80cda6d1726906dbf";

export default node;
