/**
 * @generated SignedSource<<a534f193da8b33007d02c64b6976d3a0>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type PolicyCheckStatus = "CANCELED" | "CREATED" | "ERRORED" | "OVERRIDDEN" | "PASSED" | "PENDING" | "QUEUED" | "RUNNING" | "SKIPPED" | "SOFT_FAILED" | "%future added value";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type RunGateStatus = "APPROVED" | "CANCELED" | "OVERRIDDEN" | "PENDING" | "%future added value";
export type RunStatus = "applied" | "apply_queued" | "apply_queuing" | "applying" | "canceled" | "discarded" | "errored" | "pending" | "plan_queued" | "plan_queuing" | "planned" | "planned_and_finished" | "planning" | "post_apply_running" | "post_plan_awaiting_decision" | "post_plan_running" | "pre_apply_awaiting_decision" | "pre_apply_completed" | "pre_apply_queuing" | "pre_apply_running" | "pre_plan_awaiting_decision" | "pre_plan_completed" | "pre_plan_queuing" | "pre_plan_running" | "%future added value";
export type RunTaskStageName = "POST_APPLY" | "POST_PLAN" | "PRE_APPLY" | "PRE_PLAN" | "%future added value";
export type RunTaskStageStatus = "AWAITING_OVERRIDE" | "CANCELED" | "COMPLETED" | "CREATED" | "ERRORED" | "PENDING" | "RUNNING" | "SKIPPED" | "%future added value";
export type OverrideRunGateInput = {
  clientMutationId?: string | null | undefined;
  comment?: string | null | undefined;
  gateId: string;
};
export type RunTaskStageOverrideRunButtonMutation$variables = {
  input: OverrideRunGateInput;
};
export type RunTaskStageOverrideRunButtonMutation$data = {
  readonly overrideRunGate: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly runGate: {
      readonly id: string;
      readonly metadata: {
        readonly updatedAt: any;
      };
      readonly overriddenBy: string | null | undefined;
      readonly overrideComment: string;
      readonly run: {
        readonly id: string;
        readonly status: RunStatus;
        readonly taskStages: ReadonlyArray<{
          readonly policyChecks: ReadonlyArray<{
            readonly status: PolicyCheckStatus;
          }>;
          readonly stageName: RunTaskStageName;
          readonly status: RunTaskStageStatus;
        }>;
      };
      readonly status: RunGateStatus;
    } | null | undefined;
  };
};
export type RunTaskStageOverrideRunButtonMutation = {
  response: RunTaskStageOverrideRunButtonMutation$data;
  variables: RunTaskStageOverrideRunButtonMutation$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "input"
  }
],
v1 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "status",
  "storageKey": null
},
v3 = [
  {
    "alias": null,
    "args": [
      {
        "kind": "Variable",
        "name": "input",
        "variableName": "input"
      }
    ],
    "concreteType": "RunGateMutationPayload",
    "kind": "LinkedField",
    "name": "overrideRunGate",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "RunGate",
        "kind": "LinkedField",
        "name": "runGate",
        "plural": false,
        "selections": [
          (v1/*: any*/),
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
          },
          {
            "alias": null,
            "args": null,
            "concreteType": "Run",
            "kind": "LinkedField",
            "name": "run",
            "plural": false,
            "selections": [
              (v1/*: any*/),
              (v2/*: any*/),
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
                  (v2/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "PolicyCheck",
                    "kind": "LinkedField",
                    "name": "policyChecks",
                    "plural": true,
                    "selections": [
                      (v2/*: any*/)
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
      {
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
      }
    ],
    "storageKey": null
  }
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "RunTaskStageOverrideRunButtonMutation",
    "selections": (v3/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "RunTaskStageOverrideRunButtonMutation",
    "selections": (v3/*: any*/)
  },
  "params": {
    "cacheID": "3a0480a306be18b7fc02acfbad21b10f",
    "id": null,
    "metadata": {},
    "name": "RunTaskStageOverrideRunButtonMutation",
    "operationKind": "mutation",
    "text": "mutation RunTaskStageOverrideRunButtonMutation(\n  $input: OverrideRunGateInput!\n) {\n  overrideRunGate(input: $input) {\n    runGate {\n      id\n      status\n      overriddenBy\n      overrideComment\n      metadata {\n        updatedAt\n      }\n      run {\n        id\n        status\n        taskStages {\n          stageName\n          status\n          policyChecks {\n            status\n          }\n        }\n      }\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "1ee9ab9f93505731535c623ed38e24ab";

export default node;
