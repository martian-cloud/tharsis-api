/**
 * @generated SignedSource<<8fb5f0cbf159c85c77f4059c8ed4a7c2>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ApplyStatus = "canceled" | "created" | "errored" | "finished" | "pending" | "queued" | "running" | "skipped" | "%future added value";
export type PlanStatus = "canceled" | "created" | "errored" | "finished" | "pending" | "queued" | "running" | "skipped" | "%future added value";
export type RunStatus = "applied" | "apply_queued" | "apply_queuing" | "applying" | "canceled" | "discarded" | "errored" | "pending" | "plan_queued" | "plan_queuing" | "planned" | "planned_and_finished" | "planning" | "post_apply_running" | "post_plan_awaiting_decision" | "post_plan_running" | "pre_apply_awaiting_decision" | "pre_apply_completed" | "pre_apply_queuing" | "pre_apply_running" | "pre_plan_awaiting_decision" | "pre_plan_completed" | "pre_plan_queuing" | "pre_plan_running" | "%future added value";
export type RunTaskStageName = "POST_APPLY" | "POST_PLAN" | "PRE_APPLY" | "PRE_PLAN" | "%future added value";
export type RunTaskStageStatus = "AWAITING_OVERRIDE" | "CANCELED" | "COMPLETED" | "CREATED" | "ERRORED" | "PENDING" | "RUNNING" | "SKIPPED" | "%future added value";
export type RunDetailsQuery$variables = {
  id: string;
};
export type RunDetailsQuery$data = {
  readonly run: {
    readonly apply: {
      readonly status: ApplyStatus;
    } | null | undefined;
    readonly plan: {
      readonly status: PlanStatus;
    };
    readonly status: RunStatus;
    readonly taskStages: ReadonlyArray<{
      readonly stageName: RunTaskStageName;
      readonly status: RunTaskStageStatus;
    }>;
    readonly workspace: {
      readonly fullPath: string;
      readonly locked: boolean;
    };
    readonly " $fragmentSpreads": FragmentRefs<"RunDetailsApplyStageFragment_apply" | "RunDetailsPlanStageFragment_plan" | "RunDetailsRunTaskStageFragment_taskStage" | "RunDetailsSidebarFragment_details">;
  } | null | undefined;
};
export type RunDetailsQuery = {
  response: RunDetailsQuery$data;
  variables: RunDetailsQuery$variables;
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
  "name": "status",
  "storageKey": null
},
v3 = [
  (v2/*: any*/)
],
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "stageName",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "fullPath",
  "storageKey": null
},
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "locked",
  "storageKey": null
},
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v8 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "createdAt",
  "storageKey": null
},
v9 = {
  "alias": null,
  "args": null,
  "concreteType": "ResourceMetadata",
  "kind": "LinkedField",
  "name": "metadata",
  "plural": false,
  "selections": [
    (v8/*: any*/)
  ],
  "storageKey": null
},
v10 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "cancelRequested",
  "storageKey": null
},
v11 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "runningAt",
  "storageKey": null
},
v12 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "finishedAt",
  "storageKey": null
},
v13 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "runnerAvailabilityStatus",
  "storageKey": null
},
v14 = {
  "alias": null,
  "args": null,
  "concreteType": "Workspace",
  "kind": "LinkedField",
  "name": "workspace",
  "plural": false,
  "selections": [
    (v5/*: any*/),
    (v7/*: any*/)
  ],
  "storageKey": null
},
v15 = {
  "alias": null,
  "args": null,
  "concreteType": "Job",
  "kind": "LinkedField",
  "name": "currentJob",
  "plural": false,
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "runnerPath",
      "storageKey": null
    },
    (v10/*: any*/),
    (v7/*: any*/),
    (v2/*: any*/),
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
          "name": "queuedAt",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "pendingAt",
          "storageKey": null
        },
        (v11/*: any*/),
        (v12/*: any*/)
      ],
      "storageKey": null
    },
    (v13/*: any*/),
    (v14/*: any*/),
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "outdatedJobProtocolVersion",
      "storageKey": null
    }
  ],
  "storageKey": null
},
v16 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "errorMessage",
  "storageKey": null
},
v17 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "name",
  "storageKey": null
},
v18 = {
  "alias": null,
  "args": null,
  "concreteType": "TerraformCheckResult",
  "kind": "LinkedField",
  "name": "checkResults",
  "plural": true,
  "selections": [
    (v17/*: any*/),
    (v2/*: any*/),
    {
      "alias": null,
      "args": null,
      "concreteType": "TerraformCheckResultObject",
      "kind": "LinkedField",
      "name": "objects",
      "plural": true,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "address",
          "storageKey": null
        },
        (v2/*: any*/),
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "failureMessages",
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "storageKey": null
},
v19 = {
  "alias": null,
  "args": [
    {
      "kind": "Literal",
      "name": "first",
      "value": 0
    }
  ],
  "concreteType": "JobConnection",
  "kind": "LinkedField",
  "name": "jobs",
  "plural": false,
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "totalCount",
      "storageKey": null
    }
  ],
  "storageKey": "jobs(first:0)"
},
v20 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "username",
  "storageKey": null
},
v21 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "email",
  "storageKey": null
},
v22 = [
  (v7/*: any*/),
  (v17/*: any*/),
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "resourcePath",
    "storageKey": null
  }
],
v23 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "createdBy",
  "storageKey": null
},
v24 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "messages",
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "RunDetailsQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "Run",
        "kind": "LinkedField",
        "name": "run",
        "plural": false,
        "selections": [
          (v2/*: any*/),
          {
            "alias": null,
            "args": null,
            "concreteType": "Plan",
            "kind": "LinkedField",
            "name": "plan",
            "plural": false,
            "selections": (v3/*: any*/),
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "concreteType": "RunTaskStage",
            "kind": "LinkedField",
            "name": "taskStages",
            "plural": true,
            "selections": [
              (v4/*: any*/),
              (v2/*: any*/)
            ],
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "concreteType": "Apply",
            "kind": "LinkedField",
            "name": "apply",
            "plural": false,
            "selections": (v3/*: any*/),
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
              (v5/*: any*/),
              (v6/*: any*/)
            ],
            "storageKey": null
          },
          {
            "args": null,
            "kind": "FragmentSpread",
            "name": "RunDetailsSidebarFragment_details"
          },
          {
            "args": null,
            "kind": "FragmentSpread",
            "name": "RunDetailsPlanStageFragment_plan"
          },
          {
            "args": null,
            "kind": "FragmentSpread",
            "name": "RunDetailsRunTaskStageFragment_taskStage"
          },
          {
            "args": null,
            "kind": "FragmentSpread",
            "name": "RunDetailsApplyStageFragment_apply"
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
    "name": "RunDetailsQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "Run",
        "kind": "LinkedField",
        "name": "run",
        "plural": false,
        "selections": [
          (v2/*: any*/),
          {
            "alias": null,
            "args": null,
            "concreteType": "Plan",
            "kind": "LinkedField",
            "name": "plan",
            "plural": false,
            "selections": [
              (v2/*: any*/),
              (v7/*: any*/),
              (v9/*: any*/),
              (v15/*: any*/),
              (v16/*: any*/),
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "hasChanges",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "diffSize",
                "storageKey": null
              },
              (v18/*: any*/),
              (v19/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "PlanSummary",
                "kind": "LinkedField",
                "name": "summary",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "resourceAdditions",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "resourceChanges",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "resourceDestructions",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "resourceImports",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "resourceDrift",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "outputAdditions",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "outputChanges",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "outputDestructions",
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
            "concreteType": "RunTaskStage",
            "kind": "LinkedField",
            "name": "taskStages",
            "plural": true,
            "selections": [
              (v4/*: any*/),
              (v2/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "PolicyCheck",
                "kind": "LinkedField",
                "name": "policyChecks",
                "plural": true,
                "selections": [
                  (v2/*: any*/),
                  (v4/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "Job",
                    "kind": "LinkedField",
                    "name": "currentJob",
                    "plural": false,
                    "selections": [
                      (v10/*: any*/),
                      (v7/*: any*/),
                      (v13/*: any*/),
                      (v14/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "JobTimestamps",
                        "kind": "LinkedField",
                        "name": "timestamps",
                        "plural": false,
                        "selections": [
                          (v11/*: any*/),
                          (v12/*: any*/)
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
                    "name": "checkType",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "RunGate",
                    "kind": "LinkedField",
                    "name": "runGate",
                    "plural": false,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "RunGateApprovalRule",
                        "kind": "LinkedField",
                        "name": "approvalRules",
                        "plural": true,
                        "selections": [
                          (v17/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "requiredApprovals",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": "User",
                            "kind": "LinkedField",
                            "name": "allowedUsers",
                            "plural": true,
                            "selections": [
                              (v7/*: any*/),
                              (v20/*: any*/),
                              (v21/*: any*/)
                            ],
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": "ServiceAccount",
                            "kind": "LinkedField",
                            "name": "allowedServiceAccounts",
                            "plural": true,
                            "selections": (v22/*: any*/),
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": "Team",
                            "kind": "LinkedField",
                            "name": "allowedTeams",
                            "plural": true,
                            "selections": [
                              (v7/*: any*/),
                              (v17/*: any*/)
                            ],
                            "storageKey": null
                          }
                        ],
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "RunGateApproval",
                        "kind": "LinkedField",
                        "name": "approvals",
                        "plural": true,
                        "selections": [
                          (v7/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "decision",
                            "storageKey": null
                          },
                          (v23/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "coveredRules",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": "User",
                            "kind": "LinkedField",
                            "name": "user",
                            "plural": false,
                            "selections": [
                              (v7/*: any*/),
                              (v21/*: any*/),
                              (v20/*: any*/)
                            ],
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": "ServiceAccount",
                            "kind": "LinkedField",
                            "name": "serviceAccount",
                            "plural": false,
                            "selections": (v22/*: any*/),
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "comment",
                            "storageKey": null
                          },
                          (v9/*: any*/)
                        ],
                        "storageKey": null
                      },
                      (v7/*: any*/),
                      (v2/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "overriddenBy",
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "overrideComment",
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
                            "name": "updatedAt",
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
                    "concreteType": "PolicyCheckPolicy",
                    "kind": "LinkedField",
                    "name": "policies",
                    "plural": true,
                    "selections": [
                      (v7/*: any*/),
                      (v2/*: any*/),
                      (v17/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "description",
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "PolicyCheckOPAData",
                        "kind": "LinkedField",
                        "name": "opaData",
                        "plural": false,
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "packageSource",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "packageVersionConstraint",
                            "storageKey": null
                          }
                        ],
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "PolicyCheckModuleAttestationData",
                        "kind": "LinkedField",
                        "name": "moduleAttestationData",
                        "plural": false,
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "publicKey",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "predicateType",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "verifyStateLineage",
                            "storageKey": null
                          }
                        ],
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "enforcementLevel",
                        "storageKey": null
                      },
                      (v24/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "PolicyCheckPolicyProvenance",
                        "kind": "LinkedField",
                        "name": "provenance",
                        "plural": false,
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "policyTrn",
                            "storageKey": null
                          }
                        ],
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "Policy",
                        "kind": "LinkedField",
                        "name": "policy",
                        "plural": false,
                        "selections": [
                          (v7/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "groupPath",
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
                    "name": "nodePath",
                    "storageKey": null
                  },
                  (v19/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "PolicyCheckMessagesSummary",
                    "kind": "LinkedField",
                    "name": "messagesSummary",
                    "plural": false,
                    "selections": [
                      (v24/*: any*/),
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
              }
            ],
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "concreteType": "Apply",
            "kind": "LinkedField",
            "name": "apply",
            "plural": false,
            "selections": [
              (v2/*: any*/),
              (v7/*: any*/),
              (v9/*: any*/),
              (v15/*: any*/),
              (v16/*: any*/),
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "triggeredBy",
                "storageKey": null
              },
              (v19/*: any*/)
            ],
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
              (v5/*: any*/),
              (v6/*: any*/),
              (v7/*: any*/)
            ],
            "storageKey": null
          },
          (v7/*: any*/),
          (v23/*: any*/),
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
            "kind": "ScalarField",
            "name": "assessment",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "autoApply",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "hasAdvisoryFailures",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "moduleSource",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "moduleVersion",
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
              (v8/*: any*/),
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
          {
            "alias": null,
            "args": null,
            "concreteType": "ConfigurationVersion",
            "kind": "LinkedField",
            "name": "configurationVersion",
            "plural": false,
            "selections": [
              (v7/*: any*/)
            ],
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "concreteType": "RunVariable",
            "kind": "LinkedField",
            "name": "variables",
            "plural": true,
            "selections": [
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "key",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "category",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "namespacePath",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "includedInTfConfig",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "value",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "sensitive",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "versionId",
                "storageKey": null
              }
            ],
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "forceCancelAvailableAt",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "concreteType": "StateVersion",
            "kind": "LinkedField",
            "name": "stateVersion",
            "plural": false,
            "selections": [
              {
                "alias": null,
                "args": null,
                "concreteType": "StateVersionInventory",
                "kind": "LinkedField",
                "name": "inventory",
                "plural": false,
                "selections": [
                  (v18/*: any*/)
                ],
                "storageKey": null
              },
              (v7/*: any*/)
            ],
            "storageKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "87faca690fcfc2de72cd3501a6e2b543",
    "id": null,
    "metadata": {},
    "name": "RunDetailsQuery",
    "operationKind": "query",
    "text": "query RunDetailsQuery(\n  $id: String!\n) {\n  run(id: $id) {\n    status\n    plan {\n      status\n      id\n    }\n    taskStages {\n      stageName\n      status\n    }\n    apply {\n      status\n      id\n    }\n    workspace {\n      fullPath\n      locked\n      id\n    }\n    ...RunDetailsSidebarFragment_details\n    ...RunDetailsPlanStageFragment_plan\n    ...RunDetailsRunTaskStageFragment_taskStage\n    ...RunDetailsApplyStageFragment_apply\n    id\n  }\n}\n\nfragment CheckResultsPanelFragment_checkResult on TerraformCheckResult {\n  name\n  status\n  objects {\n    address\n    status\n    failureMessages\n  }\n}\n\nfragment ForceCancelRunAlertFragment_run on Run {\n  forceCancelAvailableAt\n  ...ForceCancelRunButtonFragment_run\n}\n\nfragment ForceCancelRunButtonFragment_run on Run {\n  id\n  workspace {\n    fullPath\n    id\n  }\n}\n\nfragment NoRunnerAlertFragment_job on Job {\n  runnerAvailabilityStatus\n  workspace {\n    fullPath\n    id\n  }\n}\n\nfragment OutdatedProtocolAlertFragment_job on Job {\n  outdatedJobProtocolVersion\n}\n\nfragment RunDetailsApplyStageFragment_apply on Run {\n  id\n  status\n  plan {\n    status\n    ...RunDetailsPlanSummaryFragment_plan\n    id\n  }\n  apply {\n    metadata {\n      createdAt\n    }\n    status\n    errorMessage\n    triggeredBy\n    currentJob {\n      id\n      status\n      cancelRequested\n      timestamps {\n        queuedAt\n        pendingAt\n        runningAt\n        finishedAt\n      }\n      ...NoRunnerAlertFragment_job\n      ...OutdatedProtocolAlertFragment_job\n    }\n    jobs(first: 0) {\n      totalCount\n    }\n    id\n  }\n  ...RunVariablesFragment_variables\n  ...ForceCancelRunAlertFragment_run\n  stateVersion {\n    inventory {\n      checkResults {\n        ...CheckResultsPanelFragment_checkResult\n      }\n    }\n    ...StateVersionFileFragment_stateVersion\n    id\n  }\n}\n\nfragment RunDetailsPlanStageFragment_plan on Run {\n  id\n  status\n  createdBy\n  plan {\n    id\n    metadata {\n      createdAt\n    }\n    status\n    errorMessage\n    hasChanges\n    diffSize\n    checkResults {\n      ...CheckResultsPanelFragment_checkResult\n    }\n    currentJob {\n      id\n      status\n      cancelRequested\n      timestamps {\n        queuedAt\n        pendingAt\n        runningAt\n        finishedAt\n      }\n      ...NoRunnerAlertFragment_job\n      ...OutdatedProtocolAlertFragment_job\n    }\n    jobs(first: 0) {\n      totalCount\n    }\n    ...RunDetailsPlanSummaryFragment_plan\n  }\n  apply {\n    status\n    id\n  }\n  ...RunVariablesFragment_variables\n  ...ForceCancelRunAlertFragment_run\n}\n\nfragment RunDetailsPlanSummaryFragment_plan on Plan {\n  summary {\n    resourceAdditions\n    resourceChanges\n    resourceDestructions\n    resourceImports\n    resourceDrift\n    outputAdditions\n    outputChanges\n    outputDestructions\n  }\n}\n\nfragment RunDetailsRunTaskStageFragment_taskStage on Run {\n  id\n  createdBy\n  metadata {\n    createdAt\n  }\n  ...ForceCancelRunAlertFragment_run\n  taskStages {\n    stageName\n    status\n    ...RunTaskStageStatusPanelFragment_taskStage\n    policyChecks {\n      checkType\n      status\n      currentJob {\n        cancelRequested\n        ...NoRunnerAlertFragment_job\n        id\n      }\n      runGate {\n        approvalRules {\n          name\n        }\n        ...RunTaskStagePolicyCheckPolicyCardFragment_gate\n        id\n      }\n      policies {\n        id\n        status\n        ...RunTaskStagePolicyCheckPolicyCardFragment_policy\n      }\n      ...RunTaskStagePolicyCheckPanelFragment_check\n    }\n  }\n}\n\nfragment RunDetailsSidebarFragment_details on Run {\n  id\n  status\n  createdBy\n  isDestroy\n  assessment\n  autoApply\n  hasAdvisoryFailures\n  moduleSource\n  moduleVersion\n  workspace {\n    fullPath\n    id\n  }\n  metadata {\n    createdAt\n    trn\n  }\n  configurationVersion {\n    id\n  }\n  plan {\n    status\n    metadata {\n      createdAt\n    }\n    currentJob {\n      runnerPath\n      cancelRequested\n      id\n    }\n    id\n  }\n  taskStages {\n    stageName\n    status\n    policyChecks {\n      status\n      stageName\n    }\n  }\n  apply {\n    status\n    metadata {\n      createdAt\n    }\n    currentJob {\n      runnerPath\n      cancelRequested\n      id\n    }\n    id\n  }\n}\n\nfragment RunTaskStagePolicyApproversBoxFragment_gate on RunGate {\n  approvalRules {\n    name\n    requiredApprovals\n    allowedUsers {\n      id\n      username\n      email\n    }\n    allowedServiceAccounts {\n      id\n      name\n      resourcePath\n    }\n    allowedTeams {\n      id\n      name\n    }\n  }\n  approvals {\n    id\n    decision\n    createdBy\n    coveredRules\n    user {\n      id\n      email\n    }\n    serviceAccount {\n      id\n      name\n      resourcePath\n    }\n  }\n}\n\nfragment RunTaskStagePolicyCheckPanelFragment_check on PolicyCheck {\n  checkType\n  status\n  nodePath\n  currentJob {\n    id\n    timestamps {\n      runningAt\n      finishedAt\n    }\n  }\n  jobs(first: 0) {\n    totalCount\n  }\n  policies {\n    id\n    status\n    enforcementLevel\n  }\n  messagesSummary {\n    messages\n    truncated\n  }\n  runGate {\n    id\n    status\n    overriddenBy\n    overrideComment\n    metadata {\n      updatedAt\n    }\n    approvalRules {\n      name\n      requiredApprovals\n    }\n    approvals {\n      id\n      decision\n      comment\n      createdBy\n      coveredRules\n      metadata {\n        createdAt\n      }\n      user {\n        id\n        username\n        email\n      }\n      serviceAccount {\n        id\n        name\n        resourcePath\n      }\n    }\n  }\n}\n\nfragment RunTaskStagePolicyCheckPolicyCardFragment_gate on RunGate {\n  ...RunTaskStagePolicyApproversBoxFragment_gate\n}\n\nfragment RunTaskStagePolicyCheckPolicyCardFragment_policy on PolicyCheckPolicy {\n  id\n  name\n  description\n  opaData {\n    packageSource\n    packageVersionConstraint\n  }\n  moduleAttestationData {\n    publicKey\n    predicateType\n    verifyStateLineage\n  }\n  enforcementLevel\n  status\n  messages\n  provenance {\n    policyTrn\n  }\n  policy {\n    id\n    groupPath\n  }\n}\n\nfragment RunTaskStageStatusPanelFragment_taskStage on RunTaskStage {\n  stageName\n  status\n  policyChecks {\n    currentJob {\n      cancelRequested\n      id\n    }\n  }\n}\n\nfragment RunVariableListItemFragment_variable on RunVariable {\n  key\n  category\n  value\n  namespacePath\n  sensitive\n  versionId\n  includedInTfConfig\n}\n\nfragment RunVariablesFragment_variables on Run {\n  variables {\n    key\n    category\n    namespacePath\n    includedInTfConfig\n    ...RunVariableListItemFragment_variable\n  }\n}\n\nfragment StateVersionFileFragment_stateVersion on StateVersion {\n  id\n}\n"
  }
};
})();

(node as any).hash = "6cebed0d1a49f6cd2988daebcc66dada";

export default node;
