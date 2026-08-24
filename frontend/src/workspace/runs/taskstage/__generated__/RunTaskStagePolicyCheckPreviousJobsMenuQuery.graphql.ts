/**
 * @generated SignedSource<<380b2766e14ce62fdef9f16bea63cbba>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type RunTaskStagePolicyCheckPreviousJobsMenuQuery$variables = {
  after?: string | null | undefined;
  first: number;
  nodePath: string;
  runId: string;
};
export type RunTaskStagePolicyCheckPreviousJobsMenuQuery$data = {
  readonly " $fragmentSpreads": FragmentRefs<"RunTaskStagePolicyCheckPreviousJobsMenuFragment_jobs">;
};
export type RunTaskStagePolicyCheckPreviousJobsMenuQuery = {
  response: RunTaskStagePolicyCheckPreviousJobsMenuQuery$data;
  variables: RunTaskStagePolicyCheckPreviousJobsMenuQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "after"
},
v1 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "first"
},
v2 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "nodePath"
},
v3 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "runId"
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "__typename",
  "storageKey": null
},
v5 = [
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
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v7 = [
  (v6/*: any*/)
];
return {
  "fragment": {
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/),
      (v2/*: any*/),
      (v3/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "RunTaskStagePolicyCheckPreviousJobsMenuQuery",
    "selections": [
      {
        "args": null,
        "kind": "FragmentSpread",
        "name": "RunTaskStagePolicyCheckPreviousJobsMenuFragment_jobs"
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [
      (v3/*: any*/),
      (v2/*: any*/),
      (v1/*: any*/),
      (v0/*: any*/)
    ],
    "kind": "Operation",
    "name": "RunTaskStagePolicyCheckPreviousJobsMenuQuery",
    "selections": [
      {
        "alias": null,
        "args": [
          {
            "kind": "Variable",
            "name": "nodePath",
            "variableName": "nodePath"
          },
          {
            "kind": "Variable",
            "name": "runId",
            "variableName": "runId"
          }
        ],
        "concreteType": null,
        "kind": "LinkedField",
        "name": "runNode",
        "plural": false,
        "selections": [
          (v4/*: any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              {
                "alias": null,
                "args": (v5/*: any*/),
                "concreteType": "JobConnection",
                "kind": "LinkedField",
                "name": "jobs",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "JobEdge",
                    "kind": "LinkedField",
                    "name": "edges",
                    "plural": true,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "Job",
                        "kind": "LinkedField",
                        "name": "node",
                        "plural": false,
                        "selections": [
                          (v6/*: any*/),
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
                          },
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": "JobTimestamps",
                            "kind": "LinkedField",
                            "name": "timestamps",
                            "plural": false,
                            "selections": [
                              {
                                "alias": null,
                                "args": null,
                                "kind": "ScalarField",
                                "name": "runningAt",
                                "storageKey": null
                              },
                              {
                                "alias": null,
                                "args": null,
                                "kind": "ScalarField",
                                "name": "finishedAt",
                                "storageKey": null
                              }
                            ],
                            "storageKey": null
                          },
                          (v4/*: any*/)
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
                  }
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": (v5/*: any*/),
                "filters": [
                  "sort"
                ],
                "handle": "connection",
                "key": "RunTaskStagePolicyCheckPreviousJobsMenu_jobs",
                "kind": "LinkedHandle",
                "name": "jobs"
              }
            ],
            "type": "PolicyCheck",
            "abstractKey": null
          },
          {
            "kind": "InlineFragment",
            "selections": (v7/*: any*/),
            "type": "Apply",
            "abstractKey": null
          },
          {
            "kind": "InlineFragment",
            "selections": (v7/*: any*/),
            "type": "Plan",
            "abstractKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "c3880bd409db785010a5bbb8c2b920fa",
    "id": null,
    "metadata": {},
    "name": "RunTaskStagePolicyCheckPreviousJobsMenuQuery",
    "operationKind": "query",
    "text": "query RunTaskStagePolicyCheckPreviousJobsMenuQuery(\n  $runId: String!\n  $nodePath: String!\n  $first: Int!\n  $after: String\n) {\n  ...RunTaskStagePolicyCheckPreviousJobsMenuFragment_jobs\n}\n\nfragment RunTaskStagePolicyCheckPreviousJobsMenuFragment_jobs on Query {\n  runNode(runId: $runId, nodePath: $nodePath) {\n    __typename\n    ... on PolicyCheck {\n      jobs(first: $first, after: $after, sort: CREATED_AT_DESC) {\n        edges {\n          node {\n            id\n            ...RunTaskStagePolicyCheckPreviousJobsMenu_job\n            __typename\n          }\n          cursor\n        }\n        pageInfo {\n          endCursor\n          hasNextPage\n        }\n      }\n    }\n    ... on Apply {\n      id\n    }\n    ... on Plan {\n      id\n    }\n  }\n}\n\nfragment RunTaskStagePolicyCheckPreviousJobsMenu_job on Job {\n  id\n  status\n  metadata {\n    createdAt\n  }\n  timestamps {\n    runningAt\n    finishedAt\n  }\n}\n"
  }
};
})();

(node as any).hash = "8acabf901aae04e638110dfc4c04d473";

export default node;
