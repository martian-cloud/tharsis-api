/**
 * @generated SignedSource<<cd2a3989768ef9b344274e5e357801d6>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest, Query } from 'relay-runtime';
export type VCSProviderType = "github" | "gitlab" | "%future added value";
export type NewVCSProviderLinkQuery$variables = {
  first: number;
  fullPath: string;
  search: string;
};
export type NewVCSProviderLinkQuery$data = {
  readonly group: {
    readonly vcsProviders: {
      readonly edges: ReadonlyArray<{
        readonly node: {
          readonly autoCreateWebhooks: boolean;
          readonly id: string;
          readonly type: VCSProviderType;
        } | null;
      } | null> | null;
    };
  } | null;
};
export type NewVCSProviderLinkQuery = {
  response: NewVCSProviderLinkQuery$data;
  variables: NewVCSProviderLinkQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "first"
},
v1 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "fullPath"
},
v2 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "search"
},
v3 = [
  {
    "kind": "Variable",
    "name": "fullPath",
    "variableName": "fullPath"
  }
],
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": [
    {
      "kind": "Variable",
      "name": "first",
      "variableName": "first"
    },
    {
      "kind": "Variable",
      "name": "search",
      "variableName": "search"
    }
  ],
  "concreteType": "VCSProviderConnection",
  "kind": "LinkedField",
  "name": "vcsProviders",
  "plural": false,
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "VCSProviderEdge",
      "kind": "LinkedField",
      "name": "edges",
      "plural": true,
      "selections": [
        {
          "alias": null,
          "args": null,
          "concreteType": "VCSProvider",
          "kind": "LinkedField",
          "name": "node",
          "plural": false,
          "selections": [
            (v4/*: any*/),
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "type",
              "storageKey": null
            },
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "autoCreateWebhooks",
              "storageKey": null
            }
          ],
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "storageKey": null
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
    "name": "NewVCSProviderLinkQuery",
    "selections": [
      {
        "alias": null,
        "args": (v3/*: any*/),
        "concreteType": "Group",
        "kind": "LinkedField",
        "name": "group",
        "plural": false,
        "selections": [
          (v5/*: any*/)
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
      (v1/*: any*/),
      (v0/*: any*/),
      (v2/*: any*/)
    ],
    "kind": "Operation",
    "name": "NewVCSProviderLinkQuery",
    "selections": [
      {
        "alias": null,
        "args": (v3/*: any*/),
        "concreteType": "Group",
        "kind": "LinkedField",
        "name": "group",
        "plural": false,
        "selections": [
          (v5/*: any*/),
          (v4/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "858510ea7395aa24ee4c65dce1039d10",
    "id": null,
    "metadata": {},
    "name": "NewVCSProviderLinkQuery",
    "operationKind": "query",
    "text": "query NewVCSProviderLinkQuery(\n  $fullPath: String!\n  $first: Int!\n  $search: String!\n) {\n  group(fullPath: $fullPath) {\n    vcsProviders(first: $first, search: $search) {\n      edges {\n        node {\n          id\n          type\n          autoCreateWebhooks\n        }\n      }\n    }\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "add1cdb0ffe4074037db4b1b94d2d4b3";

export default node;
