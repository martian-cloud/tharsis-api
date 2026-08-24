/**
 * @generated SignedSource<<bc309cba41bb8fb252ba6904f895ab66>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type RunTaskStagePolicyCheckPreviousJobsMenuPaginationQuery$variables = {
  after?: string | null | undefined;
  first?: number | null | undefined;
  nodePath: string;
  runId: string;
};
export type RunTaskStagePolicyCheckPreviousJobsMenuPaginationQuery$data = {
  readonly " $fragmentSpreads": FragmentRefs<"RunTaskStagePolicyCheckPreviousJobsMenuFragment_jobs">;
};
export type RunTaskStagePolicyCheckPreviousJobsMenuPaginationQuery = {
  response: RunTaskStagePolicyCheckPreviousJobsMenuPaginationQuery$data;
  variables: RunTaskStagePolicyCheckPreviousJobsMenuPaginationQuery$variables;
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
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "nodePath"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "runId"
  }
],
v1 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "__typename",
  "storageKey": null
},
v2 = [
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
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v4 = [
  (v3/*: any*/)
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "RunTaskStagePolicyCheckPreviousJobsMenuPaginationQuery",
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
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "RunTaskStagePolicyCheckPreviousJobsMenuPaginationQuery",
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
          (v1/*: any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              {
                "alias": null,
                "args": (v2/*: any*/),
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
                          (v3/*: any*/),
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
                          (v1/*: any*/)
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
                "args": (v2/*: any*/),
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
            "selections": (v4/*: any*/),
            "type": "Apply",
            "abstractKey": null
          },
          {
            "kind": "InlineFragment",
            "selections": (v4/*: any*/),
            "type": "Plan",
            "abstractKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "d3410c2e49ab9e698e48bfd993f7e0e4",
    "id": null,
    "metadata": {},
    "name": "RunTaskStagePolicyCheckPreviousJobsMenuPaginationQuery",
    "operationKind": "query",
    "text": "query RunTaskStagePolicyCheckPreviousJobsMenuPaginationQuery(\n  $after: String\n  $first: Int\n  $nodePath: String!\n  $runId: String!\n) {\n  ...RunTaskStagePolicyCheckPreviousJobsMenuFragment_jobs\n}\n\nfragment RunTaskStagePolicyCheckPreviousJobsMenuFragment_jobs on Query {\n  runNode(runId: $runId, nodePath: $nodePath) {\n    __typename\n    ... on PolicyCheck {\n      jobs(first: $first, after: $after, sort: CREATED_AT_DESC) {\n        edges {\n          node {\n            id\n            ...RunTaskStagePolicyCheckPreviousJobsMenu_job\n            __typename\n          }\n          cursor\n        }\n        pageInfo {\n          endCursor\n          hasNextPage\n        }\n      }\n    }\n    ... on Apply {\n      id\n    }\n    ... on Plan {\n      id\n    }\n  }\n}\n\nfragment RunTaskStagePolicyCheckPreviousJobsMenu_job on Job {\n  id\n  status\n  metadata {\n    createdAt\n  }\n  timestamps {\n    runningAt\n    finishedAt\n  }\n}\n"
  }
};
})();

(node as any).hash = "8af478c43d0825a5e8b4f4a1a5252326";

export default node;
