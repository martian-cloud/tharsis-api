/**
 * @generated SignedSource<<c72559615198944f10acf6f8a365687f>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type JobStatus = "canceled" | "canceling" | "failed" | "finished" | "pending" | "queued" | "running" | "%future added value";
export type JobType = "apply" | "opa" | "plan" | "%future added value";
export type RunStatus = "applied" | "apply_queued" | "apply_queuing" | "applying" | "canceled" | "discarded" | "errored" | "pending" | "plan_queued" | "plan_queuing" | "planned" | "planned_and_finished" | "planning" | "post_apply_running" | "post_plan_awaiting_decision" | "post_plan_running" | "pre_apply_awaiting_decision" | "pre_apply_completed" | "pre_apply_queuing" | "pre_apply_running" | "pre_plan_awaiting_decision" | "pre_plan_completed" | "pre_plan_queuing" | "pre_plan_running" | "%future added value";
export type RunnerType = "group" | "shared" | "%future added value";
export type JobDetailsQuery$variables = {
  id: string;
};
export type JobDetailsQuery$data = {
  readonly node: {
    readonly forceCanceled?: boolean;
    readonly id?: string;
    readonly metadata?: {
      readonly createdAt: any;
      readonly trn: string;
    };
    readonly run?: {
      readonly createdByActor: {
        readonly " $fragmentSpreads": FragmentRefs<"ActorDisplayFragment_actor">;
      } | null | undefined;
      readonly hasAdvisoryFailures: boolean;
      readonly id: string;
      readonly status: RunStatus;
      readonly workspace: {
        readonly fullPath: string;
      };
    };
    readonly runner?: {
      readonly groupPath: string;
      readonly id: string;
      readonly name: string;
      readonly type: RunnerType;
    } | null | undefined;
    readonly runnerPath?: string | null | undefined;
    readonly status?: JobStatus;
    readonly tags?: ReadonlyArray<string>;
    readonly timestamps?: {
      readonly finishedAt: any | null | undefined;
      readonly pendingAt: any | null | undefined;
      readonly runningAt: any | null | undefined;
    };
    readonly type?: JobType;
    readonly workspace?: {
      readonly fullPath: string;
    };
    readonly " $fragmentSpreads": FragmentRefs<"JobDetailsResourceUsageCardFragment_job" | "OutdatedProtocolAlertFragment_job">;
  } | null | undefined;
};
export type JobDetailsQuery = {
  response: JobDetailsQuery$data;
  variables: JobDetailsQuery$variables;
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
  "kind": "ScalarField",
  "name": "status",
  "storageKey": null
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "type",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "tags",
  "storageKey": null
},
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "forceCanceled",
  "storageKey": null
},
v7 = {
  "alias": null,
  "args": null,
  "concreteType": "Runner",
  "kind": "LinkedField",
  "name": "runner",
  "plural": false,
  "selections": [
    (v2/*: any*/),
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "name",
      "storageKey": null
    },
    (v4/*: any*/),
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "groupPath",
      "storageKey": null
    }
  ],
  "storageKey": null
},
v8 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "runnerPath",
  "storageKey": null
},
v9 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "fullPath",
  "storageKey": null
},
v10 = {
  "alias": null,
  "args": null,
  "concreteType": "Workspace",
  "kind": "LinkedField",
  "name": "workspace",
  "plural": false,
  "selections": [
    (v9/*: any*/)
  ],
  "storageKey": null
},
v11 = {
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
v12 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "hasAdvisoryFailures",
  "storageKey": null
},
v13 = {
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
      "name": "pendingAt",
      "storageKey": null
    },
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
v14 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "__typename",
  "storageKey": null
},
v15 = {
  "alias": null,
  "args": null,
  "concreteType": "Workspace",
  "kind": "LinkedField",
  "name": "workspace",
  "plural": false,
  "selections": [
    (v9/*: any*/),
    (v2/*: any*/)
  ],
  "storageKey": null
},
v16 = [
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "trnSnapshot",
    "storageKey": null
  }
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "JobDetailsQuery",
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
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "OutdatedProtocolAlertFragment_job"
              },
              (v7/*: any*/),
              (v8/*: any*/),
              (v10/*: any*/),
              (v11/*: any*/),
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
                    "concreteType": null,
                    "kind": "LinkedField",
                    "name": "createdByActor",
                    "plural": false,
                    "selections": [
                      {
                        "args": null,
                        "kind": "FragmentSpread",
                        "name": "ActorDisplayFragment_actor"
                      }
                    ],
                    "storageKey": null
                  },
                  (v3/*: any*/),
                  (v12/*: any*/),
                  (v10/*: any*/)
                ],
                "storageKey": null
              },
              (v13/*: any*/),
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "JobDetailsResourceUsageCardFragment_job"
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
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "JobDetailsQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          (v14/*: any*/),
          (v2/*: any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              (v3/*: any*/),
              (v4/*: any*/),
              (v5/*: any*/),
              (v6/*: any*/),
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "outdatedJobProtocolVersion",
                "storageKey": null
              },
              (v7/*: any*/),
              (v8/*: any*/),
              (v15/*: any*/),
              (v11/*: any*/),
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
                    "concreteType": null,
                    "kind": "LinkedField",
                    "name": "createdByActor",
                    "plural": false,
                    "selections": [
                      (v14/*: any*/),
                      {
                        "kind": "TypeDiscriminator",
                        "abstractKey": "__isActor"
                      },
                      (v4/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "principalId",
                        "storageKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": "User",
                            "kind": "LinkedField",
                            "name": "user",
                            "plural": false,
                            "selections": [
                              {
                                "alias": null,
                                "args": null,
                                "kind": "ScalarField",
                                "name": "email",
                                "storageKey": null
                              },
                              (v2/*: any*/)
                            ],
                            "storageKey": null
                          }
                        ],
                        "type": "UserActor",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v16/*: any*/),
                        "type": "ServiceAccountActor",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v16/*: any*/),
                        "type": "VCSProviderActor",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v16/*: any*/),
                        "type": "JobActor",
                        "abstractKey": null
                      }
                    ],
                    "storageKey": null
                  },
                  (v3/*: any*/),
                  (v12/*: any*/),
                  (v15/*: any*/)
                ],
                "storageKey": null
              },
              (v13/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "JobResourceUsageMetrics",
                "kind": "LinkedField",
                "name": "resourceUsageMetrics",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "totalCpuTimeMs",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "peakMemoryBytes",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "totalNetworkReceivedBytes",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "totalNetworkSentBytes",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "totalDiskReadBytes",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "totalDiskWriteBytes",
                    "storageKey": null
                  }
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "JobResourceUsageLimits",
                "kind": "LinkedField",
                "name": "resourceUsageLimits",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "memoryBytes",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "networkReceivedBytes",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "networkSentBytes",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "diskReadBytes",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "diskWriteBytes",
                    "storageKey": null
                  }
                ],
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
    "cacheID": "ae705d03c3a434198918c59daa7a1377",
    "id": null,
    "metadata": {},
    "name": "JobDetailsQuery",
    "operationKind": "query",
    "text": "query JobDetailsQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ... on Job {\n      id\n      status\n      type\n      tags\n      forceCanceled\n      ...OutdatedProtocolAlertFragment_job\n      runner {\n        id\n        name\n        type\n        groupPath\n      }\n      runnerPath\n      workspace {\n        fullPath\n        id\n      }\n      metadata {\n        createdAt\n        trn\n      }\n      run {\n        id\n        createdByActor {\n          __typename\n          ...ActorDisplayFragment_actor\n        }\n        status\n        hasAdvisoryFailures\n        workspace {\n          fullPath\n          id\n        }\n      }\n      timestamps {\n        pendingAt\n        runningAt\n        finishedAt\n      }\n      ...JobDetailsResourceUsageCardFragment_job\n    }\n    id\n  }\n}\n\nfragment ActorDisplayFragment_actor on Actor {\n  __isActor: __typename\n  type\n  principalId\n  ... on UserActor {\n    user {\n      email\n      id\n    }\n  }\n  ... on ServiceAccountActor {\n    trnSnapshot\n  }\n  ... on VCSProviderActor {\n    trnSnapshot\n  }\n  ... on JobActor {\n    trnSnapshot\n  }\n}\n\nfragment JobDetailsResourceUsageCardFragment_job on Job {\n  resourceUsageMetrics {\n    totalCpuTimeMs\n  }\n  ...JobResourceUsageMetricsFragment_job\n}\n\nfragment JobResourceUsageMetricsFragment_job on Job {\n  resourceUsageMetrics {\n    peakMemoryBytes\n    totalCpuTimeMs\n    totalNetworkReceivedBytes\n    totalNetworkSentBytes\n    totalDiskReadBytes\n    totalDiskWriteBytes\n  }\n  resourceUsageLimits {\n    memoryBytes\n    networkReceivedBytes\n    networkSentBytes\n    diskReadBytes\n    diskWriteBytes\n  }\n}\n\nfragment OutdatedProtocolAlertFragment_job on Job {\n  outdatedJobProtocolVersion\n}\n"
  }
};
})();

(node as any).hash = "c60877c4f7dc811bef0ab0d856903043";

export default node;
