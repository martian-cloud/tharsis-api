/**
 * @generated SignedSource<<c65545d3e45a0763ddccbab81a0a29f6>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type CancelRunInput = {
  clientMutationId?: string | null | undefined;
  comment?: string | null | undefined;
  force?: boolean | null | undefined;
  runId: string;
};
export type RunTaskStageCancelRunButtonMutation$variables = {
  input: CancelRunInput;
};
export type RunTaskStageCancelRunButtonMutation$data = {
  readonly cancelRun: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly run: {
      readonly " $fragmentSpreads": FragmentRefs<"RunDetailsRunTaskStageFragment_taskStage">;
    } | null | undefined;
  };
};
export type RunTaskStageCancelRunButtonMutation = {
  response: RunTaskStageCancelRunButtonMutation$data;
  variables: RunTaskStageCancelRunButtonMutation$variables;
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
  "name": "type",
  "storageKey": null
},
v3 = {
  "alias": null,
  "args": null,
  "concreteType": "Problem",
  "kind": "LinkedField",
  "name": "problems",
  "plural": true,
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "message",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "field",
      "storageKey": null
    },
    (v2/*: any*/)
  ],
  "storageKey": null
},
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
  "name": "email",
  "storageKey": null
},
v6 = [
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "trnSnapshot",
    "storageKey": null
  }
],
v7 = [
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "__typename",
    "storageKey": null
  },
  {
    "kind": "TypeDiscriminator",
    "abstractKey": "__isActor"
  },
  (v2/*: any*/),
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
          (v5/*: any*/),
          (v4/*: any*/)
        ],
        "storageKey": null
      }
    ],
    "type": "UserActor",
    "abstractKey": null
  },
  {
    "kind": "InlineFragment",
    "selections": (v6/*: any*/),
    "type": "ServiceAccountActor",
    "abstractKey": null
  },
  {
    "kind": "InlineFragment",
    "selections": (v6/*: any*/),
    "type": "VCSProviderActor",
    "abstractKey": null
  },
  {
    "kind": "InlineFragment",
    "selections": (v6/*: any*/),
    "type": "JobActor",
    "abstractKey": null
  }
],
v8 = {
  "alias": null,
  "args": null,
  "concreteType": null,
  "kind": "LinkedField",
  "name": "createdByActor",
  "plural": false,
  "selections": (v7/*: any*/),
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
v10 = {
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
    (v4/*: any*/)
  ],
  "storageKey": null
},
v11 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "status",
  "storageKey": null
},
v12 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "name",
  "storageKey": null
},
v13 = [
  (v4/*: any*/),
  (v12/*: any*/),
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "resourcePath",
    "storageKey": null
  }
],
v14 = {
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
    "name": "RunTaskStageCancelRunButtonMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "RunMutationPayload",
        "kind": "LinkedField",
        "name": "cancelRun",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "Run",
            "kind": "LinkedField",
            "name": "run",
            "plural": false,
            "selections": [
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "RunDetailsRunTaskStageFragment_taskStage"
              }
            ],
            "storageKey": null
          },
          (v3/*: any*/)
        ],
        "storageKey": null
      }
    ],
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "RunTaskStageCancelRunButtonMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "RunMutationPayload",
        "kind": "LinkedField",
        "name": "cancelRun",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "Run",
            "kind": "LinkedField",
            "name": "run",
            "plural": false,
            "selections": [
              (v4/*: any*/),
              (v8/*: any*/),
              (v9/*: any*/),
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "forceCancelAvailableAt",
                "storageKey": null
              },
              (v10/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "RunTaskStage",
                "kind": "LinkedField",
                "name": "taskStages",
                "plural": true,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "stageName",
                    "storageKey": null
                  },
                  (v11/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "PolicyCheck",
                    "kind": "LinkedField",
                    "name": "policyChecks",
                    "plural": true,
                    "selections": [
                      {
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
                            "name": "cancelRequested",
                            "storageKey": null
                          },
                          (v4/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "runnerAvailabilityStatus",
                            "storageKey": null
                          },
                          (v10/*: any*/),
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
                      (v11/*: any*/),
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
                              (v12/*: any*/),
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
                                  (v4/*: any*/),
                                  {
                                    "alias": null,
                                    "args": null,
                                    "kind": "ScalarField",
                                    "name": "username",
                                    "storageKey": null
                                  },
                                  (v5/*: any*/)
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
                                "selections": (v13/*: any*/),
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
                                  (v4/*: any*/),
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
                            "concreteType": "RunGateApproval",
                            "kind": "LinkedField",
                            "name": "approvals",
                            "plural": true,
                            "selections": [
                              (v4/*: any*/),
                              {
                                "alias": null,
                                "args": null,
                                "kind": "ScalarField",
                                "name": "decision",
                                "storageKey": null
                              },
                              (v8/*: any*/),
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
                                  (v4/*: any*/),
                                  (v5/*: any*/)
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
                                "selections": (v13/*: any*/),
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
                          (v4/*: any*/),
                          (v11/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": null,
                            "kind": "LinkedField",
                            "name": "overriddenByActor",
                            "plural": false,
                            "selections": (v7/*: any*/),
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
                          (v4/*: any*/),
                          (v11/*: any*/),
                          (v12/*: any*/),
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
                          (v14/*: any*/),
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
                      {
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
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "PolicyCheckMessagesSummary",
                        "kind": "LinkedField",
                        "name": "messagesSummary",
                        "plural": false,
                        "selections": [
                          (v14/*: any*/),
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
              }
            ],
            "storageKey": null
          },
          (v3/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "42497a0762d50d7661bea69fe7fab615",
    "id": null,
    "metadata": {},
    "name": "RunTaskStageCancelRunButtonMutation",
    "operationKind": "mutation",
    "text": "mutation RunTaskStageCancelRunButtonMutation(\n  $input: CancelRunInput!\n) {\n  cancelRun(input: $input) {\n    run {\n      ...RunDetailsRunTaskStageFragment_taskStage\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment ActorDisplayFragment_actor on Actor {\n  __isActor: __typename\n  type\n  principalId\n  ... on UserActor {\n    user {\n      email\n      id\n    }\n  }\n  ... on ServiceAccountActor {\n    trnSnapshot\n  }\n  ... on VCSProviderActor {\n    trnSnapshot\n  }\n  ... on JobActor {\n    trnSnapshot\n  }\n}\n\nfragment ForceCancelRunAlertFragment_run on Run {\n  forceCancelAvailableAt\n  ...ForceCancelRunButtonFragment_run\n}\n\nfragment ForceCancelRunButtonFragment_run on Run {\n  id\n  workspace {\n    fullPath\n    id\n  }\n}\n\nfragment NoRunnerAlertFragment_job on Job {\n  runnerAvailabilityStatus\n  workspace {\n    fullPath\n    id\n  }\n}\n\nfragment RunDetailsRunTaskStageFragment_taskStage on Run {\n  id\n  createdByActor {\n    __typename\n    ...ActorDisplayFragment_actor\n  }\n  metadata {\n    createdAt\n  }\n  ...ForceCancelRunAlertFragment_run\n  taskStages {\n    stageName\n    status\n    ...RunTaskStageStatusPanelFragment_taskStage\n    policyChecks {\n      checkType\n      status\n      currentJob {\n        cancelRequested\n        ...NoRunnerAlertFragment_job\n        id\n      }\n      runGate {\n        approvalRules {\n          name\n        }\n        ...RunTaskStagePolicyCheckPolicyCardFragment_gate\n        id\n      }\n      policies {\n        id\n        status\n        ...RunTaskStagePolicyCheckPolicyCardFragment_policy\n      }\n      ...RunTaskStagePolicyCheckPanelFragment_check\n    }\n  }\n}\n\nfragment RunTaskStagePolicyApproversBoxFragment_gate on RunGate {\n  approvalRules {\n    name\n    requiredApprovals\n    allowedUsers {\n      id\n      username\n      email\n    }\n    allowedServiceAccounts {\n      id\n      name\n      resourcePath\n    }\n    allowedTeams {\n      id\n      name\n    }\n  }\n  approvals {\n    id\n    decision\n    createdByActor {\n      __typename\n      ...ActorDisplayFragment_actor\n    }\n    coveredRules\n    user {\n      id\n      email\n    }\n    serviceAccount {\n      id\n      name\n      resourcePath\n    }\n  }\n}\n\nfragment RunTaskStagePolicyCheckPanelFragment_check on PolicyCheck {\n  checkType\n  status\n  nodePath\n  currentJob {\n    id\n    timestamps {\n      runningAt\n      finishedAt\n    }\n  }\n  jobs(first: 0) {\n    totalCount\n  }\n  policies {\n    id\n    status\n    enforcementLevel\n  }\n  messagesSummary {\n    messages\n    truncated\n  }\n  runGate {\n    id\n    status\n    overriddenByActor {\n      __typename\n      ...ActorDisplayFragment_actor\n    }\n    overrideComment\n    metadata {\n      updatedAt\n    }\n    approvalRules {\n      name\n      requiredApprovals\n    }\n    approvals {\n      id\n      decision\n      comment\n      createdByActor {\n        __typename\n        ...ActorDisplayFragment_actor\n      }\n      coveredRules\n      metadata {\n        createdAt\n      }\n    }\n  }\n}\n\nfragment RunTaskStagePolicyCheckPolicyCardFragment_gate on RunGate {\n  ...RunTaskStagePolicyApproversBoxFragment_gate\n}\n\nfragment RunTaskStagePolicyCheckPolicyCardFragment_policy on PolicyCheckPolicy {\n  id\n  name\n  description\n  opaData {\n    packageSource\n    packageVersionConstraint\n  }\n  moduleAttestationData {\n    publicKey\n    predicateType\n    verifyStateLineage\n  }\n  enforcementLevel\n  status\n  messages\n  provenance {\n    policyTrn\n  }\n  policy {\n    id\n    groupPath\n  }\n}\n\nfragment RunTaskStageStatusPanelFragment_taskStage on RunTaskStage {\n  stageName\n  status\n  policyChecks {\n    currentJob {\n      cancelRequested\n      id\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "e24afeea7d8bb55438a15730ffb9ab04";

export default node;
