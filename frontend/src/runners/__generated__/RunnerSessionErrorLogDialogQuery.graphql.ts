/**
 * @generated SignedSource<<d50ce93de9da824ce56e9403640428c0>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type RunnerSessionErrorLogDialogQuery$variables = {
  id: string;
  limit: number;
  startOffset: number;
};
export type RunnerSessionErrorLogDialogQuery$data = {
  readonly node: {
    readonly errorCount?: number;
    readonly errorLog?: {
      readonly data: string;
      readonly lastUpdatedAt: any | null | undefined;
      readonly size: number;
    } | null | undefined;
    readonly id?: string;
  } | null | undefined;
};
export type RunnerSessionErrorLogDialogQuery = {
  response: RunnerSessionErrorLogDialogQuery$data;
  variables: RunnerSessionErrorLogDialogQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "id"
},
v1 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "limit"
},
v2 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "startOffset"
},
v3 = [
  {
    "kind": "Variable",
    "name": "id",
    "variableName": "id"
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
  "args": null,
  "kind": "ScalarField",
  "name": "errorCount",
  "storageKey": null
},
v6 = {
  "alias": null,
  "args": null,
  "concreteType": "RunnerSessionErrorLog",
  "kind": "LinkedField",
  "name": "errorLog",
  "plural": false,
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "lastUpdatedAt",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "size",
      "storageKey": null
    },
    {
      "alias": null,
      "args": [
        {
          "kind": "Variable",
          "name": "limit",
          "variableName": "limit"
        },
        {
          "kind": "Variable",
          "name": "startOffset",
          "variableName": "startOffset"
        }
      ],
      "kind": "ScalarField",
      "name": "data",
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
    "name": "RunnerSessionErrorLogDialogQuery",
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
            "kind": "InlineFragment",
            "selections": [
              (v4/*: any*/),
              (v5/*: any*/),
              (v6/*: any*/)
            ],
            "type": "RunnerSession",
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
    "argumentDefinitions": [
      (v0/*: any*/),
      (v2/*: any*/),
      (v1/*: any*/)
    ],
    "kind": "Operation",
    "name": "RunnerSessionErrorLogDialogQuery",
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
            "kind": "InlineFragment",
            "selections": [
              (v5/*: any*/),
              (v6/*: any*/)
            ],
            "type": "RunnerSession",
            "abstractKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "4888684dbad3c953ffb6843c3bafac48",
    "id": null,
    "metadata": {},
    "name": "RunnerSessionErrorLogDialogQuery",
    "operationKind": "query",
    "text": "query RunnerSessionErrorLogDialogQuery(\n  $id: String!\n  $startOffset: Int!\n  $limit: Int!\n) {\n  node(id: $id) {\n    __typename\n    ... on RunnerSession {\n      id\n      errorCount\n      errorLog {\n        lastUpdatedAt\n        size\n        data(startOffset: $startOffset, limit: $limit)\n      }\n    }\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "e8c7d5968e00459995fa07a9867f2afe";

export default node;
