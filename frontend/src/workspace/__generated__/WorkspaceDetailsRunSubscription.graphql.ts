/**
 * @generated SignedSource<<3518eae4a2254fffc2d7c6415144ab7b>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type RunSubscriptionInput = {
  ancestorGroupId?: string | null | undefined;
  runId?: string | null | undefined;
  workspaceId?: string | null | undefined;
  workspacePath?: string | null | undefined;
};
export type WorkspaceDetailsRunSubscription$variables = {
  input: RunSubscriptionInput;
};
export type WorkspaceDetailsRunSubscription$data = {
  readonly workspaceRunEvents: {
    readonly action: string;
    readonly run: {
      readonly id: string;
      readonly " $fragmentSpreads": FragmentRefs<"RunDetailsApplyStageFragment_apply" | "RunDetailsPlanStageFragment_plan" | "RunDetailsRunTaskStageFragment_taskStage" | "RunDetailsSidebarFragment_details" | "RunListItemFragment_run">;
    };
  };
};
export type WorkspaceDetailsRunSubscription = {
  response: WorkspaceDetailsRunSubscription$data;
  variables: WorkspaceDetailsRunSubscription$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "input"
  }
],
v1 = [
  {
    "kind": "Variable",
    "name": "input",
    "variableName": "input"
  }
],
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "action",
  "storageKey": null
},
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "createdAt",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "createdBy",
  "storageKey": null
},
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "status",
  "storageKey": null
},
v7 = {
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
    (v3/*: any*/)
  ],
  "storageKey": null
},
v8 = {
  "alias": null,
  "args": null,
  "concreteType": "ResourceMetadata",
  "kind": "LinkedField",
  "name": "metadata",
  "plural": false,
  "selections": [
    (v4/*: any*/)
  ],
  "storageKey": null
},
v9 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "cancelRequested",
  "storageKey": null
},
v10 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "runningAt",
  "storageKey": null
},
v11 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "finishedAt",
  "storageKey": null
},
v12 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "runnerAvailabilityStatus",
  "storageKey": null
},
v13 = {
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
    (v9/*: any*/),
    (v3/*: any*/),
    (v6/*: any*/),
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
        (v10/*: any*/),
        (v11/*: any*/)
      ],
      "storageKey": null
    },
    (v12/*: any*/),
    (v7/*: any*/),
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
v14 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "errorMessage",
  "storageKey": null
},
v15 = {
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
v16 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "name",
  "storageKey": null
},
v17 = {
  "alias": null,
  "args": null,
  "concreteType": "TerraformCheckResult",
  "kind": "LinkedField",
  "name": "checkResults",
  "plural": true,
  "selections": [
    (v16/*: any*/),
    (v6/*: any*/),
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
        (v6/*: any*/),
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
v18 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "stageName",
  "storageKey": null
},
v19 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "username",
  "storageKey": null
},
v20 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "email",
  "storageKey": null
},
v21 = [
  (v3/*: any*/),
  (v16/*: any*/),
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "resourcePath",
    "storageKey": null
  }
],
v22 = {
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
    "name": "WorkspaceDetailsRunSubscription",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "RunEvent",
        "kind": "LinkedField",
        "name": "workspaceRunEvents",
        "plural": false,
        "selections": [
          (v2/*: any*/),
          {
            "alias": null,
            "args": null,
            "concreteType": "Run",
            "kind": "LinkedField",
            "name": "run",
            "plural": false,
            "selections": [
              (v3/*: any*/),
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "RunListItemFragment_run"
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
        "storageKey": null
      }
    ],
    "type": "Subscription",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "WorkspaceDetailsRunSubscription",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "RunEvent",
        "kind": "LinkedField",
        "name": "workspaceRunEvents",
        "plural": false,
        "selections": [
          (v2/*: any*/),
          {
            "alias": null,
            "args": null,
            "concreteType": "Run",
            "kind": "LinkedField",
            "name": "run",
            "plural": false,
            "selections": [
              (v3/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "ResourceMetadata",
                "kind": "LinkedField",
                "name": "metadata",
                "plural": false,
                "selections": [
                  (v4/*: any*/),
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
              (v5/*: any*/),
              (v6/*: any*/),
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
                "name": "hasAdvisoryFailures",
                "storageKey": null
              },
              (v7/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "Apply",
                "kind": "LinkedField",
                "name": "apply",
                "plural": false,
                "selections": [
                  (v6/*: any*/),
                  (v3/*: any*/),
                  (v8/*: any*/),
                  (v13/*: any*/),
                  (v14/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "triggeredBy",
                    "storageKey": null
                  },
                  (v15/*: any*/)
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "Plan",
                "kind": "LinkedField",
                "name": "plan",
                "plural": false,
                "selections": [
                  (v6/*: any*/),
                  (v3/*: any*/),
                  (v8/*: any*/),
                  (v13/*: any*/),
                  (v14/*: any*/),
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
                  (v17/*: any*/),
                  (v15/*: any*/),
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
                  (v18/*: any*/),
                  (v6/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "PolicyCheck",
                    "kind": "LinkedField",
                    "name": "policyChecks",
                    "plural": true,
                    "selections": [
                      (v6/*: any*/),
                      (v18/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "Job",
                        "kind": "LinkedField",
                        "name": "currentJob",
                        "plural": false,
                        "selections": [
                          (v9/*: any*/),
                          (v3/*: any*/),
                          (v12/*: any*/),
                          (v7/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": "JobTimestamps",
                            "kind": "LinkedField",
                            "name": "timestamps",
                            "plural": false,
                            "selections": [
                              (v10/*: any*/),
                              (v11/*: any*/)
                            ],
                            "storageKey": null
                          }
                        ],
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
                              (v16/*: any*/),
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
                                  (v3/*: any*/),
                                  (v19/*: any*/),
                                  (v20/*: any*/)
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
                                "selections": (v21/*: any*/),
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
                                  (v3/*: any*/),
                                  (v16/*: any*/)
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
                              (v3/*: any*/),
                              {
                                "alias": null,
                                "args": null,
                                "kind": "ScalarField",
                                "name": "decision",
                                "storageKey": null
                              },
                              (v5/*: any*/),
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
                                  (v3/*: any*/),
                                  (v20/*: any*/),
                                  (v19/*: any*/)
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
                                "selections": (v21/*: any*/),
                                "storageKey": null
                              },
                              {
                                "alias": null,
                                "args": null,
                                "kind": "ScalarField",
                                "name": "comment",
                                "storageKey": null
                              },
                              (v8/*: any*/)
                            ],
                            "storageKey": null
                          },
                          (v3/*: any*/),
                          (v6/*: any*/),
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
                          (v3/*: any*/),
                          (v6/*: any*/),
                          (v16/*: any*/),
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
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "enforcementLevel",
                            "storageKey": null
                          },
                          (v22/*: any*/),
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
                              (v3/*: any*/),
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
                        "name": "checkType",
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "nodePath",
                        "storageKey": null
                      },
                      (v15/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "PolicyCheckMessagesSummary",
                        "kind": "LinkedField",
                        "name": "messagesSummary",
                        "plural": false,
                        "selections": [
                          (v22/*: any*/),
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
                "kind": "ScalarField",
                "name": "autoApply",
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
                "concreteType": "ConfigurationVersion",
                "kind": "LinkedField",
                "name": "configurationVersion",
                "plural": false,
                "selections": [
                  (v3/*: any*/)
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
                      (v17/*: any*/)
                    ],
                    "storageKey": null
                  },
                  (v3/*: any*/)
                ],
                "storageKey": null
              }
            ],
            "storageKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "106acfdfed3b735d56ac27be64713539",
    "id": null,
    "metadata": {},
    "name": "WorkspaceDetailsRunSubscription",
    "operationKind": "subscription",
    "text": "subscription WorkspaceDetailsRunSubscription(\n  $input: RunSubscriptionInput!\n) {\n  workspaceRunEvents(input: $input) {\n    action\n    run {\n      id\n      ...RunListItemFragment_run\n      ...RunDetailsSidebarFragment_details\n      ...RunDetailsPlanStageFragment_plan\n      ...RunDetailsRunTaskStageFragment_taskStage\n      ...RunDetailsApplyStageFragment_apply\n    }\n  }\n}\n\nfragment CheckResultsPanelFragment_checkResult on TerraformCheckResult {\n  name\n  status\n  objects {\n    address\n    status\n    failureMessages\n  }\n}\n\nfragment ForceCancelRunAlertFragment_run on Run {\n  forceCancelAvailableAt\n  ...ForceCancelRunButtonFragment_run\n}\n\nfragment ForceCancelRunButtonFragment_run on Run {\n  id\n  workspace {\n    fullPath\n    id\n  }\n}\n\nfragment NoRunnerAlertFragment_job on Job {\n  runnerAvailabilityStatus\n  workspace {\n    fullPath\n    id\n  }\n}\n\nfragment OutdatedProtocolAlertFragment_job on Job {\n  outdatedJobProtocolVersion\n}\n\nfragment RunDetailsApplyStageFragment_apply on Run {\n  id\n  status\n  plan {\n    status\n    ...RunDetailsPlanSummaryFragment_plan\n    id\n  }\n  apply {\n    metadata {\n      createdAt\n    }\n    status\n    errorMessage\n    triggeredBy\n    currentJob {\n      id\n      status\n      cancelRequested\n      timestamps {\n        queuedAt\n        pendingAt\n        runningAt\n        finishedAt\n      }\n      ...NoRunnerAlertFragment_job\n      ...OutdatedProtocolAlertFragment_job\n    }\n    jobs(first: 0) {\n      totalCount\n    }\n    id\n  }\n  ...RunVariablesFragment_variables\n  ...ForceCancelRunAlertFragment_run\n  stateVersion {\n    inventory {\n      checkResults {\n        ...CheckResultsPanelFragment_checkResult\n      }\n    }\n    ...StateVersionFileFragment_stateVersion\n    id\n  }\n}\n\nfragment RunDetailsPlanStageFragment_plan on Run {\n  id\n  status\n  createdBy\n  plan {\n    id\n    metadata {\n      createdAt\n    }\n    status\n    errorMessage\n    hasChanges\n    diffSize\n    checkResults {\n      ...CheckResultsPanelFragment_checkResult\n    }\n    currentJob {\n      id\n      status\n      cancelRequested\n      timestamps {\n        queuedAt\n        pendingAt\n        runningAt\n        finishedAt\n      }\n      ...NoRunnerAlertFragment_job\n      ...OutdatedProtocolAlertFragment_job\n    }\n    jobs(first: 0) {\n      totalCount\n    }\n    ...RunDetailsPlanSummaryFragment_plan\n  }\n  apply {\n    status\n    id\n  }\n  ...RunVariablesFragment_variables\n  ...ForceCancelRunAlertFragment_run\n}\n\nfragment RunDetailsPlanSummaryFragment_plan on Plan {\n  summary {\n    resourceAdditions\n    resourceChanges\n    resourceDestructions\n    resourceImports\n    resourceDrift\n    outputAdditions\n    outputChanges\n    outputDestructions\n  }\n}\n\nfragment RunDetailsRunTaskStageFragment_taskStage on Run {\n  id\n  createdBy\n  metadata {\n    createdAt\n  }\n  ...ForceCancelRunAlertFragment_run\n  taskStages {\n    stageName\n    status\n    ...RunTaskStageStatusPanelFragment_taskStage\n    policyChecks {\n      currentJob {\n        cancelRequested\n        ...NoRunnerAlertFragment_job\n        id\n      }\n      runGate {\n        approvalRules {\n          name\n        }\n        ...RunTaskStagePolicyCheckPolicyCardFragment_gate\n        id\n      }\n      policies {\n        id\n        status\n        ...RunTaskStagePolicyCheckPolicyCardFragment_policy\n      }\n      ...RunTaskStagePolicyCheckPanelFragment_check\n    }\n  }\n}\n\nfragment RunDetailsSidebarFragment_details on Run {\n  id\n  status\n  createdBy\n  isDestroy\n  assessment\n  autoApply\n  hasAdvisoryFailures\n  moduleSource\n  moduleVersion\n  workspace {\n    fullPath\n    id\n  }\n  metadata {\n    createdAt\n    trn\n  }\n  configurationVersion {\n    id\n  }\n  plan {\n    status\n    metadata {\n      createdAt\n    }\n    currentJob {\n      runnerPath\n      cancelRequested\n      id\n    }\n    id\n  }\n  taskStages {\n    stageName\n    status\n    policyChecks {\n      status\n      stageName\n    }\n  }\n  apply {\n    status\n    metadata {\n      createdAt\n    }\n    currentJob {\n      runnerPath\n      cancelRequested\n      id\n    }\n    id\n  }\n}\n\nfragment RunListItemFragment_run on Run {\n  metadata {\n    createdAt\n    trn\n  }\n  id\n  createdBy\n  status\n  isDestroy\n  assessment\n  hasAdvisoryFailures\n  workspace {\n    fullPath\n    id\n  }\n  apply {\n    status\n    id\n  }\n  ...RunStageIconsFragment_run\n}\n\nfragment RunStageIconsFragment_run on Run {\n  id\n  status\n  hasAdvisoryFailures\n  plan {\n    status\n    id\n  }\n  taskStages {\n    stageName\n    status\n  }\n  apply {\n    status\n    id\n  }\n  workspace {\n    fullPath\n    id\n  }\n}\n\nfragment RunTaskStagePolicyApproversBoxFragment_gate on RunGate {\n  approvalRules {\n    name\n    requiredApprovals\n    allowedUsers {\n      id\n      username\n      email\n    }\n    allowedServiceAccounts {\n      id\n      name\n      resourcePath\n    }\n    allowedTeams {\n      id\n      name\n    }\n  }\n  approvals {\n    id\n    decision\n    createdBy\n    coveredRules\n    user {\n      id\n      email\n    }\n    serviceAccount {\n      id\n      name\n      resourcePath\n    }\n  }\n}\n\nfragment RunTaskStagePolicyCheckPanelFragment_check on PolicyCheck {\n  checkType\n  status\n  nodePath\n  currentJob {\n    id\n    timestamps {\n      runningAt\n      finishedAt\n    }\n  }\n  jobs(first: 0) {\n    totalCount\n  }\n  policies {\n    id\n    status\n  }\n  messagesSummary {\n    messages\n    truncated\n  }\n  runGate {\n    id\n    status\n    overriddenBy\n    overrideComment\n    metadata {\n      updatedAt\n    }\n    approvalRules {\n      name\n      requiredApprovals\n    }\n    approvals {\n      id\n      decision\n      comment\n      createdBy\n      coveredRules\n      metadata {\n        createdAt\n      }\n      user {\n        id\n        username\n        email\n      }\n      serviceAccount {\n        id\n        name\n        resourcePath\n      }\n    }\n  }\n}\n\nfragment RunTaskStagePolicyCheckPolicyCardFragment_gate on RunGate {\n  ...RunTaskStagePolicyApproversBoxFragment_gate\n}\n\nfragment RunTaskStagePolicyCheckPolicyCardFragment_policy on PolicyCheckPolicy {\n  id\n  name\n  description\n  packageSource\n  packageVersionConstraint\n  enforcementLevel\n  status\n  messages\n  provenance {\n    policyTrn\n  }\n  policy {\n    id\n    groupPath\n  }\n}\n\nfragment RunTaskStageStatusPanelFragment_taskStage on RunTaskStage {\n  stageName\n  status\n  policyChecks {\n    currentJob {\n      cancelRequested\n      id\n    }\n  }\n}\n\nfragment RunVariableListItemFragment_variable on RunVariable {\n  key\n  category\n  value\n  namespacePath\n  sensitive\n  versionId\n  includedInTfConfig\n}\n\nfragment RunVariablesFragment_variables on Run {\n  variables {\n    key\n    category\n    namespacePath\n    includedInTfConfig\n    ...RunVariableListItemFragment_variable\n  }\n}\n\nfragment StateVersionFileFragment_stateVersion on StateVersion {\n  id\n}\n"
  }
};
})();

(node as any).hash = "9ebf1c360128513ec572f795c4dd47f4";

export default node;
