/**
 * @generated SignedSource<<d392c7e225aedce349385598f11c0e40>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ApprovalsPaginationQuery$variables = {
  after?: string | null | undefined;
  first?: number | null | undefined;
};
export type ApprovalsPaginationQuery$data = {
  readonly " $fragmentSpreads": FragmentRefs<"ApprovalsFragment_gates">;
};
export type ApprovalsPaginationQuery = {
  response: ApprovalsPaginationQuery$data;
  variables: ApprovalsPaginationQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "after"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "first"
  }
],
v1 = [
  {
    "kind": "Variable",
    "name": "after",
    "variableName": "after"
  },
  {
    "kind": "Variable",
    "name": "first",
    "variableName": "first"
  },
  {
    "kind": "Literal",
    "name": "sort",
    "value": "CREATED_AT_DESC"
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
  "name": "status",
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "ApprovalsPaginationQuery",
    "selections": [
      {
        "args": null,
        "kind": "FragmentSpread",
        "name": "ApprovalsFragment_gates"
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "ApprovalsPaginationQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "RunGateConnection",
        "kind": "LinkedField",
        "name": "runGatesAwaitingMyDecision",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "totalCount",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "concreteType": "RunGateEdge",
            "kind": "LinkedField",
            "name": "edges",
            "plural": true,
            "selections": [
              {
                "alias": null,
                "args": null,
                "concreteType": "RunGate",
                "kind": "LinkedField",
                "name": "node",
                "plural": false,
                "selections": [
                  (v2/*: any*/),
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
                  },
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "Run",
                    "kind": "LinkedField",
                    "name": "run",
                    "plural": false,
                    "selections": [
                      (v2/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "isDestroy",
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "Workspace",
                        "kind": "LinkedField",
                        "name": "workspace",
                        "plural": false,
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "fullPath",
                            "storageKey": null
                          },
                          (v2/*: any*/)
                        ],
                        "storageKey": null
                      }
                    ],
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "PolicyCheck",
                    "kind": "LinkedField",
                    "name": "policyCheck",
                    "plural": false,
                    "selections": [
                      (v2/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "checkType",
                        "storageKey": null
                      },
                      (v3/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "stageName",
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "PolicyCheckPolicy",
                        "kind": "LinkedField",
                        "name": "policies",
                        "plural": true,
                        "selections": [
                          (v2/*: any*/),
                          (v3/*: any*/)
                        ],
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "PolicyCheckMessagesSummary",
                        "kind": "LinkedField",
                        "name": "messagesSummary",
                        "plural": false,
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "messages",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "truncated",
                            "storageKey": null
                          }
                        ],
                        "storageKey": null
                      }
                    ],
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "__typename",
                    "storageKey": null
                  }
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "cursor",
                "storageKey": null
              }
            ],
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "concreteType": "PageInfo",
            "kind": "LinkedField",
            "name": "pageInfo",
            "plural": false,
            "selections": [
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "endCursor",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "hasNextPage",
                "storageKey": null
              }
            ],
            "storageKey": null
          },
          {
            "kind": "ClientExtension",
            "selections": [
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "__id",
                "storageKey": null
              }
            ]
          }
        ],
        "storageKey": null
      },
      {
        "alias": null,
        "args": (v1/*: any*/),
        "filters": [
          "sort"
        ],
        "handle": "connection",
        "key": "Approvals_runGatesAwaitingMyDecision",
        "kind": "LinkedHandle",
        "name": "runGatesAwaitingMyDecision"
      }
    ]
  },
  "params": {
    "cacheID": "0a24ac1f7304c9982ce8ea9370049cd6",
    "id": null,
    "metadata": {},
    "name": "ApprovalsPaginationQuery",
    "operationKind": "query",
    "text": "query ApprovalsPaginationQuery(\n  $after: String\n  $first: Int\n) {\n  ...ApprovalsFragment_gates\n}\n\nfragment ApprovalGateCardFragment_gate on RunGate {\n  id\n  metadata {\n    createdAt\n  }\n  run {\n    id\n    isDestroy\n    workspace {\n      fullPath\n      id\n    }\n  }\n  policyCheck {\n    id\n    checkType\n    status\n    stageName\n    policies {\n      id\n      status\n    }\n    messagesSummary {\n      messages\n      truncated\n    }\n  }\n}\n\nfragment ApprovalsFragment_gates on Query {\n  runGatesAwaitingMyDecision(after: $after, first: $first, sort: CREATED_AT_DESC) {\n    totalCount\n    edges {\n      node {\n        id\n        ...ApprovalGateCardFragment_gate\n        __typename\n      }\n      cursor\n    }\n    pageInfo {\n      endCursor\n      hasNextPage\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "da1ef645904949c9e4bbaa984c3e4e82";

export default node;
