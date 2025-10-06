/**
 * @generated SignedSource<<0b6e55f5c60e2c0b9022a8f4de1349cf>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type ApplyRunInput = {
  clientMutationId?: string | null | undefined;
  comment?: string | null | undefined;
  runId: string;
};
export type RunDetailsApplyStageApplyRunMutation$variables = {
  input: ApplyRunInput;
};
export type RunDetailsApplyStageApplyRunMutation$data = {
  readonly applyRun: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly run: {
      readonly " $fragmentSpreads": FragmentRefs<"RunDetailsApplyStageFragment_apply">;
    } | null | undefined;
  };
};
export type RunDetailsApplyStageApplyRunMutation = {
  response: RunDetailsApplyStageApplyRunMutation$data;
  variables: RunDetailsApplyStageApplyRunMutation$variables;
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
  "name": "status",
  "storageKey": null
},
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "createdAt",
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
    (v4/*: any*/)
  ],
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "RunDetailsApplyStageApplyRunMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "RunMutationPayload",
        "kind": "LinkedField",
        "name": "applyRun",
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
                "name": "RunDetailsApplyStageFragment_apply"
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
    "name": "RunDetailsApplyStageApplyRunMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "RunMutationPayload",
        "kind": "LinkedField",
        "name": "applyRun",
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
              (v5/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "Plan",
                "kind": "LinkedField",
                "name": "plan",
                "plural": false,
                "selections": [
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
                  },
                  (v5/*: any*/),
                  (v4/*: any*/)
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
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "ResourceMetadata",
                    "kind": "LinkedField",
                    "name": "metadata",
                    "plural": false,
                    "selections": [
                      (v6/*: any*/),
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
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "errorMessage",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "triggeredBy",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "Job",
                    "kind": "LinkedField",
                    "name": "currentJob",
                    "plural": false,
                    "selections": [
                      (v4/*: any*/),
                      (v5/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "cancelRequested",
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
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "runnerAvailabilityStatus",
                        "storageKey": null
                      },
                      (v7/*: any*/),
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
                            "kind": "Literal",
                            "name": "limit",
                            "value": 51200
                          },
                          {
                            "kind": "Literal",
                            "name": "startOffset",
                            "value": 0
                          }
                        ],
                        "kind": "ScalarField",
                        "name": "logs",
                        "storageKey": "logs(limit:51200,startOffset:0)"
                      },
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "tags",
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "Runner",
                        "kind": "LinkedField",
                        "name": "runner",
                        "plural": false,
                        "selections": [
                          (v4/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "name",
                            "storageKey": null
                          },
                          (v2/*: any*/),
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
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "runnerPath",
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
                          (v6/*: any*/)
                        ],
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
              (v7/*: any*/)
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
    "cacheID": "9636f03c766e92c62b73de83103e7c91",
    "id": null,
    "metadata": {},
    "name": "RunDetailsApplyStageApplyRunMutation",
    "operationKind": "mutation",
    "text": "mutation RunDetailsApplyStageApplyRunMutation(\n  $input: ApplyRunInput!\n) {\n  applyRun(input: $input) {\n    run {\n      ...RunDetailsApplyStageFragment_apply\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment ForceCancelRunAlertFragment_run on Run {\n  forceCancelAvailableAt\n  ...ForceCancelRunButtonFragment_run\n}\n\nfragment ForceCancelRunButtonDialogFragment_run on Run {\n  workspace {\n    fullPath\n    id\n  }\n}\n\nfragment ForceCancelRunButtonFragment_run on Run {\n  id\n  ...ForceCancelRunButtonDialogFragment_run\n}\n\nfragment JobLogsFragment_logs on Job {\n  id\n  status\n  logLastUpdatedAt\n  logSize\n  logs(startOffset: 0, limit: 51200)\n}\n\nfragment NoRunnerAlertFragment_job on Job {\n  runnerAvailabilityStatus\n  workspace {\n    fullPath\n    id\n  }\n}\n\nfragment RunDetailsApplyStageFragment_apply on Run {\n  id\n  status\n  plan {\n    summary {\n      resourceAdditions\n      resourceChanges\n      resourceDestructions\n    }\n    status\n    ...RunDetailsPlanSummaryFragment_plan\n    id\n  }\n  apply {\n    metadata {\n      createdAt\n      trn\n    }\n    status\n    errorMessage\n    triggeredBy\n    currentJob {\n      id\n      status\n      cancelRequested\n      timestamps {\n        queuedAt\n        pendingAt\n        runningAt\n        finishedAt\n      }\n      ...NoRunnerAlertFragment_job\n      ...JobLogsFragment_logs\n      ...RunJobDialog_currentJob\n    }\n    id\n  }\n  ...RunVariablesFragment_variables\n  ...ForceCancelRunAlertFragment_run\n}\n\nfragment RunDetailsPlanSummaryFragment_plan on Plan {\n  summary {\n    resourceAdditions\n    resourceChanges\n    resourceDestructions\n    resourceImports\n    resourceDrift\n    outputAdditions\n    outputChanges\n    outputDestructions\n  }\n}\n\nfragment RunJobDialog_currentJob on Job {\n  id\n  status\n  tags\n  runner {\n    id\n    name\n    type\n    groupPath\n  }\n  runnerPath\n  metadata {\n    createdAt\n  }\n  timestamps {\n    pendingAt\n    runningAt\n    finishedAt\n  }\n}\n\nfragment RunVariableListItemFragment_variable on RunVariable {\n  key\n  category\n  value\n  namespacePath\n  sensitive\n  versionId\n  includedInTfConfig\n}\n\nfragment RunVariablesFragment_variables on Run {\n  variables {\n    key\n    category\n    namespacePath\n    includedInTfConfig\n    ...RunVariableListItemFragment_variable\n  }\n}\n"
  }
};
})();

(node as any).hash = "aba9b0cb7f225ecb791a2d0b2d6e88df";

export default node;
