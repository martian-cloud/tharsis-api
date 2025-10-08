/**
 * @generated SignedSource<<017680a3462cd32fcd3e927dca5c4852>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ApplyStatus = "canceled" | "created" | "errored" | "finished" | "pending" | "queued" | "running" | "%future added value";
export type PlanStatus = "canceled" | "errored" | "finished" | "pending" | "queued" | "running" | "%future added value";
export type RunStatus = "applied" | "apply_queued" | "applying" | "canceled" | "errored" | "pending" | "plan_queued" | "planned" | "planned_and_finished" | "planning" | "%future added value";
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
    readonly workspace: {
      readonly fullPath: string;
      readonly locked: boolean;
      readonly metadata: {
        readonly updatedAt: any;
      };
    };
    readonly " $fragmentSpreads": FragmentRefs<"RunDetailsApplyStageFragment_apply" | "RunDetailsPlanStageFragment_plan" | "RunDetailsSidebarFragment_details">;
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
  "name": "fullPath",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "locked",
  "storageKey": null
},
v6 = {
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
v10 = {
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
v11 = {
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
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "cancelRequested",
      "storageKey": null
    },
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
    {
      "alias": null,
      "args": null,
      "concreteType": "Workspace",
      "kind": "LinkedField",
      "name": "workspace",
      "plural": false,
      "selections": [
        (v4/*: any*/),
        (v7/*: any*/)
      ],
      "storageKey": null
    },
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
        (v7/*: any*/),
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "name",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "type",
          "storageKey": null
        },
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
    (v10/*: any*/)
  ],
  "storageKey": null
},
v12 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "errorMessage",
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
              (v4/*: any*/),
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
              (v11/*: any*/),
              (v12/*: any*/),
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
            "concreteType": "Apply",
            "kind": "LinkedField",
            "name": "apply",
            "plural": false,
            "selections": [
              (v2/*: any*/),
              (v7/*: any*/),
              (v9/*: any*/),
              (v11/*: any*/),
              (v12/*: any*/),
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "triggeredBy",
                "storageKey": null
              }
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
              (v4/*: any*/),
              (v5/*: any*/),
              (v6/*: any*/),
              (v7/*: any*/)
            ],
            "storageKey": null
          },
          (v7/*: any*/),
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "createdBy",
            "storageKey": null
          },
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
          (v10/*: any*/),
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
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "a1e19c837f0fe477c63f93384e9b2e23",
    "id": null,
    "metadata": {},
    "name": "RunDetailsQuery",
    "operationKind": "query",
    "text": "query RunDetailsQuery(\n  $id: String!\n) {\n  run(id: $id) {\n    status\n    plan {\n      status\n      id\n    }\n    apply {\n      status\n      id\n    }\n    workspace {\n      fullPath\n      locked\n      metadata {\n        updatedAt\n      }\n      id\n    }\n    ...RunDetailsSidebarFragment_details\n    ...RunDetailsPlanStageFragment_plan\n    ...RunDetailsApplyStageFragment_apply\n    id\n  }\n}\n\nfragment ForceCancelRunAlertFragment_run on Run {\n  forceCancelAvailableAt\n  ...ForceCancelRunButtonFragment_run\n}\n\nfragment ForceCancelRunButtonDialogFragment_run on Run {\n  workspace {\n    fullPath\n    id\n  }\n}\n\nfragment ForceCancelRunButtonFragment_run on Run {\n  id\n  ...ForceCancelRunButtonDialogFragment_run\n}\n\nfragment JobLogsFragment_logs on Job {\n  id\n  status\n  logLastUpdatedAt\n  logSize\n  logs(startOffset: 0, limit: 51200)\n}\n\nfragment NoRunnerAlertFragment_job on Job {\n  runnerAvailabilityStatus\n  workspace {\n    fullPath\n    id\n  }\n}\n\nfragment RunDetailsApplyStageFragment_apply on Run {\n  id\n  status\n  plan {\n    summary {\n      resourceAdditions\n      resourceChanges\n      resourceDestructions\n    }\n    status\n    ...RunDetailsPlanSummaryFragment_plan\n    id\n  }\n  apply {\n    metadata {\n      createdAt\n      trn\n    }\n    status\n    errorMessage\n    triggeredBy\n    currentJob {\n      id\n      status\n      cancelRequested\n      timestamps {\n        queuedAt\n        pendingAt\n        runningAt\n        finishedAt\n      }\n      ...NoRunnerAlertFragment_job\n      ...JobLogsFragment_logs\n      ...RunJobDialog_currentJob\n    }\n    id\n  }\n  ...RunVariablesFragment_variables\n  ...ForceCancelRunAlertFragment_run\n}\n\nfragment RunDetailsPlanStageFragment_plan on Run {\n  id\n  createdBy\n  workspace {\n    locked\n    metadata {\n      updatedAt\n    }\n    id\n  }\n  plan {\n    metadata {\n      createdAt\n      trn\n    }\n    status\n    errorMessage\n    hasChanges\n    diffSize\n    currentJob {\n      id\n      status\n      cancelRequested\n      timestamps {\n        queuedAt\n        pendingAt\n        runningAt\n        finishedAt\n      }\n      ...NoRunnerAlertFragment_job\n      ...JobLogsFragment_logs\n      ...RunJobDialog_currentJob\n    }\n    ...RunDetailsPlanSummaryFragment_plan\n    id\n  }\n  apply {\n    status\n    id\n  }\n  ...RunVariablesFragment_variables\n  ...ForceCancelRunAlertFragment_run\n}\n\nfragment RunDetailsPlanSummaryFragment_plan on Plan {\n  summary {\n    resourceAdditions\n    resourceChanges\n    resourceDestructions\n    resourceImports\n    resourceDrift\n    outputAdditions\n    outputChanges\n    outputDestructions\n  }\n}\n\nfragment RunDetailsSidebarFragment_details on Run {\n  id\n  status\n  createdBy\n  isDestroy\n  assessment\n  moduleSource\n  moduleVersion\n  workspace {\n    fullPath\n    id\n  }\n  metadata {\n    createdAt\n  }\n  configurationVersion {\n    id\n  }\n  plan {\n    status\n    metadata {\n      createdAt\n    }\n    currentJob {\n      runnerPath\n      cancelRequested\n      id\n    }\n    id\n  }\n  apply {\n    status\n    metadata {\n      createdAt\n    }\n    currentJob {\n      runnerPath\n      cancelRequested\n      id\n    }\n    id\n  }\n}\n\nfragment RunJobDialog_currentJob on Job {\n  id\n  status\n  tags\n  runner {\n    id\n    name\n    type\n    groupPath\n  }\n  runnerPath\n  metadata {\n    createdAt\n  }\n  timestamps {\n    pendingAt\n    runningAt\n    finishedAt\n  }\n}\n\nfragment RunVariableListItemFragment_variable on RunVariable {\n  key\n  category\n  value\n  namespacePath\n  sensitive\n  versionId\n  includedInTfConfig\n}\n\nfragment RunVariablesFragment_variables on Run {\n  variables {\n    key\n    category\n    namespacePath\n    includedInTfConfig\n    ...RunVariableListItemFragment_variable\n  }\n}\n"
  }
};
})();

(node as any).hash = "fbab4a1961b5c4360a18662b9472d1e6";

export default node;
