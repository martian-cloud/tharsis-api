/**
 * @generated SignedSource<<e594db2d6662517dfc1717126edcbbe8>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type AgentSessionTraceViewerRunsQuery$variables = {
  sessionId: string;
};
export type AgentSessionTraceViewerRunsQuery$data = {
  readonly node: {
    readonly runs?: {
      readonly edges: ReadonlyArray<{
        readonly node: {
          readonly id: string;
          readonly metadata: {
            readonly createdAt: any;
          };
          readonly status: string;
        } | null | undefined;
      } | null | undefined> | null | undefined;
    };
  } | null | undefined;
};
export type AgentSessionTraceViewerRunsQuery = {
  response: AgentSessionTraceViewerRunsQuery$data;
  variables: AgentSessionTraceViewerRunsQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "sessionId"
  }
],
v1 = [
  {
    "kind": "Variable",
    "name": "id",
    "variableName": "sessionId"
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
  "kind": "InlineFragment",
  "selections": [
    {
      "alias": null,
      "args": [
        {
          "kind": "Literal",
          "name": "first",
          "value": 100
        },
        {
          "kind": "Literal",
          "name": "sort",
          "value": "CREATED_AT_ASC"
        }
      ],
      "concreteType": "AgentSessionRunConnection",
      "kind": "LinkedField",
      "name": "runs",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "concreteType": "AgentSessionRunEdge",
          "kind": "LinkedField",
          "name": "edges",
          "plural": true,
          "selections": [
            {
              "alias": null,
              "args": null,
              "concreteType": "AgentSessionRun",
              "kind": "LinkedField",
              "name": "node",
              "plural": false,
              "selections": [
                (v2/*: any*/),
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
                    }
                  ],
                  "storageKey": null
                }
              ],
              "storageKey": null
            }
          ],
          "storageKey": null
        }
      ],
      "storageKey": "runs(first:100,sort:\"CREATED_AT_ASC\")"
    }
  ],
  "type": "AgentSession",
  "abstractKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "AgentSessionTraceViewerRunsQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          (v3/*: any*/)
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
    "name": "AgentSessionTraceViewerRunsQuery",
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
          (v3/*: any*/),
          (v2/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "24888db2718166ee7ef6cd6d40238bc8",
    "id": null,
    "metadata": {},
    "name": "AgentSessionTraceViewerRunsQuery",
    "operationKind": "query",
    "text": "query AgentSessionTraceViewerRunsQuery(\n  $sessionId: String!\n) {\n  node(id: $sessionId) {\n    __typename\n    ... on AgentSession {\n      runs(first: 100, sort: CREATED_AT_ASC) {\n        edges {\n          node {\n            id\n            status\n            metadata {\n              createdAt\n            }\n          }\n        }\n      }\n    }\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "68e99de31b457bef833adb616355517c";

export default node;
