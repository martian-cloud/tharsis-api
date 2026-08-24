/**
 * @generated SignedSource<<49ab3e24125db0d3f1a223b5b12baa33>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type RetryRunNodeInput = {
  clientMutationId?: string | null | undefined;
  nodePath: string;
  runId: string;
};
export type RunTaskStageRetryPolicyCheckButtonMutation$variables = {
  input: RetryRunNodeInput;
};
export type RunTaskStageRetryPolicyCheckButtonMutation$data = {
  readonly retryRunNode: {
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
export type RunTaskStageRetryPolicyCheckButtonMutation = {
  response: RunTaskStageRetryPolicyCheckButtonMutation$data;
  variables: RunTaskStageRetryPolicyCheckButtonMutation$variables;
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
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "type",
      "storageKey": null
    }
  ],
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
  "name": "createdBy",
  "storageKey": null
},
v5 = {
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
v6 = {
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
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "status",
  "storageKey": null
},
v8 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "name",
  "storageKey": null
},
v9 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "username",
  "storageKey": null
},
v10 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "email",
  "storageKey": null
},
v11 = [
  (v3/*: any*/),
  (v8/*: any*/),
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "resourcePath",
    "storageKey": null
  }
],
v12 = {
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
    "name": "RunTaskStageRetryPolicyCheckButtonMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "RunMutationPayload",
        "kind": "LinkedField",
        "name": "retryRunNode",
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
          (v2/*: any*/)
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
    "name": "RunTaskStageRetryPolicyCheckButtonMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "RunMutationPayload",
        "kind": "LinkedField",
        "name": "retryRunNode",
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
              (v3/*: any*/),
              (v4/*: any*/),
              (v5/*: any*/),
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "forceCancelAvailableAt",
                "storageKey": null
              },
              (v6/*: any*/),
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
                  (v7/*: any*/),
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
                          (v3/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "runnerAvailabilityStatus",
                            "storageKey": null
                          },
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
                              (v8/*: any*/),
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
                                  (v9/*: any*/),
                                  (v10/*: any*/)
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
                                "selections": (v11/*: any*/),
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
                                  (v8/*: any*/)
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
                              (v4/*: any*/),
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
                                  (v10/*: any*/),
                                  (v9/*: any*/)
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
                                "selections": (v11/*: any*/),
                                "storageKey": null
                              },
                              {
                                "alias": null,
                                "args": null,
                                "kind": "ScalarField",
                                "name": "comment",
                                "storageKey": null
                              },
                              (v5/*: any*/)
                            ],
                            "storageKey": null
                          },
                          (v3/*: any*/),
                          (v7/*: any*/),
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
                          (v7/*: any*/),
                          (v8/*: any*/),
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
                          (v12/*: any*/),
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
                      (v7/*: any*/),
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
                          (v12/*: any*/),
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
          (v2/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "aaba750fa248dca1486c5758ac09676a",
    "id": null,
    "metadata": {},
    "name": "RunTaskStageRetryPolicyCheckButtonMutation",
    "operationKind": "mutation",
    "text": "mutation RunTaskStageRetryPolicyCheckButtonMutation(\n  $input: RetryRunNodeInput!\n) {\n  retryRunNode(input: $input) {\n    run {\n      ...RunDetailsRunTaskStageFragment_taskStage\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment ForceCancelRunAlertFragment_run on Run {\n  forceCancelAvailableAt\n  ...ForceCancelRunButtonFragment_run\n}\n\nfragment ForceCancelRunButtonFragment_run on Run {\n  id\n  workspace {\n    fullPath\n    id\n  }\n}\n\nfragment NoRunnerAlertFragment_job on Job {\n  runnerAvailabilityStatus\n  workspace {\n    fullPath\n    id\n  }\n}\n\nfragment RunDetailsRunTaskStageFragment_taskStage on Run {\n  id\n  createdBy\n  metadata {\n    createdAt\n  }\n  ...ForceCancelRunAlertFragment_run\n  taskStages {\n    stageName\n    status\n    ...RunTaskStageStatusPanelFragment_taskStage\n    policyChecks {\n      currentJob {\n        cancelRequested\n        ...NoRunnerAlertFragment_job\n        id\n      }\n      runGate {\n        approvalRules {\n          name\n        }\n        ...RunTaskStagePolicyCheckPolicyCardFragment_gate\n        id\n      }\n      policies {\n        id\n        status\n        ...RunTaskStagePolicyCheckPolicyCardFragment_policy\n      }\n      ...RunTaskStagePolicyCheckPanelFragment_check\n    }\n  }\n}\n\nfragment RunTaskStagePolicyApproversBoxFragment_gate on RunGate {\n  approvalRules {\n    name\n    requiredApprovals\n    allowedUsers {\n      id\n      username\n      email\n    }\n    allowedServiceAccounts {\n      id\n      name\n      resourcePath\n    }\n    allowedTeams {\n      id\n      name\n    }\n  }\n  approvals {\n    id\n    decision\n    createdBy\n    coveredRules\n    user {\n      id\n      email\n    }\n    serviceAccount {\n      id\n      name\n      resourcePath\n    }\n  }\n}\n\nfragment RunTaskStagePolicyCheckPanelFragment_check on PolicyCheck {\n  checkType\n  status\n  nodePath\n  currentJob {\n    id\n    timestamps {\n      runningAt\n      finishedAt\n    }\n  }\n  jobs(first: 0) {\n    totalCount\n  }\n  policies {\n    id\n    status\n  }\n  messagesSummary {\n    messages\n    truncated\n  }\n  runGate {\n    id\n    status\n    overriddenBy\n    overrideComment\n    metadata {\n      updatedAt\n    }\n    approvalRules {\n      name\n      requiredApprovals\n    }\n    approvals {\n      id\n      decision\n      comment\n      createdBy\n      coveredRules\n      metadata {\n        createdAt\n      }\n      user {\n        id\n        username\n        email\n      }\n      serviceAccount {\n        id\n        name\n        resourcePath\n      }\n    }\n  }\n}\n\nfragment RunTaskStagePolicyCheckPolicyCardFragment_gate on RunGate {\n  ...RunTaskStagePolicyApproversBoxFragment_gate\n}\n\nfragment RunTaskStagePolicyCheckPolicyCardFragment_policy on PolicyCheckPolicy {\n  id\n  name\n  description\n  packageSource\n  packageVersionConstraint\n  enforcementLevel\n  status\n  messages\n  provenance {\n    policyTrn\n  }\n  policy {\n    id\n    groupPath\n  }\n}\n\nfragment RunTaskStageStatusPanelFragment_taskStage on RunTaskStage {\n  stageName\n  status\n  policyChecks {\n    currentJob {\n      cancelRequested\n      id\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "d4d14de6c58a165f8a784942bdb143e9";

export default node;
