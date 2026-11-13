/**
 * @generated SignedSource<<2115318b0204d840c2df872e72cc689f>>
 * @lightSyntaxTransform
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
    readonly job: {
      readonly " $fragmentSpreads": FragmentRefs<"JobLogsFragment_logs">;
    } | null | undefined;
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
      (v0/*:: as any*/),
      (v1/*:: as any*/),
      (v2/*:: as any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "JobLogsQuery",
    "selections": [
      {
        "alias": null,
        "args": (v3/*:: as any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          {
            "fragment": {
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
            },
            "kind": "AliasedInlineFragmentSpread",
            "name": "job"
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
      (v0/*:: as any*/),
      (v2/*:: as any*/),
      (v1/*:: as any*/)
    ],
    "kind": "Operation",
    "name": "JobLogsQuery",
    "selections": [
      {
        "alias": null,
        "args": (v3/*:: as any*/),
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
                "name": "type",
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
    "cacheID": "10ce6df325767ba1d34f38afcb93cc11",
    "id": null,
    "metadata": {},
    "name": "JobLogsQuery",
    "operationKind": "query",
    "text": "query JobLogsQuery(\n  $id: String!\n  $startOffset: Int!\n  $limit: Int!\n) {\n  node(id: $id) {\n    __typename\n    ... on Job {\n      ...JobLogsFragment_logs\n    }\n    id\n  }\n}\n\nfragment JobLogsFragment_logs on Job {\n  id\n  status\n  type\n  completed\n  logLastUpdatedAt\n  logSize\n  logs(startOffset: $startOffset, limit: $limit)\n}\n"
  }
};
})();

(node as any).hash = "6548c0f8adb4286534e720a6291890a8";

export default node;
