/**
 * @generated SignedSource<<27fb74d322a221d88b3cfb82837335b7>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type JobLogsQuery$variables = {
  id: string;
  limit: number;
  startOffset: number;
};
export type JobLogsQuery$data = {
  readonly node: {
    readonly " $fragmentSpreads": FragmentRefs<"JobLogsFragment_logs">;
  } | null | undefined;
};
export type JobLogsQuery = {
  response: JobLogsQuery$data;
  variables: JobLogsQuery$variables;
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
];
return {
  "fragment": {
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/),
      (v2/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "JobLogsQuery",
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
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "JobLogsFragment_logs"
              }
            ],
            "type": "Job",
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
    "name": "JobLogsQuery",
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
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "id",
            "storageKey": null
          },
          {
            "kind": "InlineFragment",
            "selections": [
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "status",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "completed",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "logLastUpdatedAt",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "logSize",
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
                "name": "logs",
                "storageKey": null
              }
            ],
            "type": "Job",
            "abstractKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "2d49c7105ba492cbfd30f1784c4a38dc",
    "id": null,
    "metadata": {},
    "name": "JobLogsQuery",
    "operationKind": "query",
    "text": "query JobLogsQuery(\n  $id: String!\n  $startOffset: Int!\n  $limit: Int!\n) {\n  node(id: $id) {\n    __typename\n    ... on Job {\n      ...JobLogsFragment_logs\n    }\n    id\n  }\n}\n\nfragment JobLogsFragment_logs on Job {\n  id\n  status\n  completed\n  logLastUpdatedAt\n  logSize\n  logs(startOffset: $startOffset, limit: $limit)\n}\n"
  }
};
})();

(node as any).hash = "50be62a82b1691241edc494ef91760d7";

export default node;
