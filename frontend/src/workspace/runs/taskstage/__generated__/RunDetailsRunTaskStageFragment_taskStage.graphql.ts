/**
 * @generated SignedSource<<835c067552310e0378ad9788afa0b8a5>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type RunTaskStageName = "POST_APPLY" | "POST_PLAN" | "PRE_APPLY" | "PRE_PLAN" | "%future added value";
export type RunTaskStageStatus = "AWAITING_OVERRIDE" | "CANCELED" | "COMPLETED" | "CREATED" | "ERRORED" | "PENDING" | "RUNNING" | "SKIPPED" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type RunDetailsRunTaskStageFragment_taskStage$data = {
  readonly createdBy: string;
  readonly id: string;
  readonly metadata: {
    readonly createdAt: any;
  };
  readonly taskStages: ReadonlyArray<{
    readonly policyChecks: ReadonlyArray<{
      readonly currentJob: {
        readonly cancelRequested: boolean;
        readonly " $fragmentSpreads": FragmentRefs<"NoRunnerAlertFragment_job">;
      } | null | undefined;
      readonly policies: ReadonlyArray<{
        readonly id: string;
        readonly status: string;
        readonly " $fragmentSpreads": FragmentRefs<"RunTaskStagePolicyCheckPolicyCardFragment_policy">;
      }>;
      readonly runGate: {
        readonly approvalRules: ReadonlyArray<{
          readonly name: string;
        }>;
        readonly " $fragmentSpreads": FragmentRefs<"RunTaskStagePolicyCheckPolicyCardFragment_gate">;
      } | null | undefined;
      readonly " $fragmentSpreads": FragmentRefs<"RunTaskStagePolicyCheckPanelFragment_check">;
    }>;
    readonly stageName: RunTaskStageName;
    readonly status: RunTaskStageStatus;
    readonly " $fragmentSpreads": FragmentRefs<"RunTaskStageStatusPanelFragment_taskStage">;
  }>;
  readonly " $fragmentSpreads": FragmentRefs<"ForceCancelRunAlertFragment_run">;
  readonly " $fragmentType": "RunDetailsRunTaskStageFragment_taskStage";
};
export type RunDetailsRunTaskStageFragment_taskStage$key = {
  readonly " $data"?: RunDetailsRunTaskStageFragment_taskStage$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunDetailsRunTaskStageFragment_taskStage">;
};

const node: ReaderFragment = (function(){
var v0 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v1 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "status",
  "storageKey": null
};
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "RunDetailsRunTaskStageFragment_taskStage",
  "selections": [
    (v0/*: any*/),
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
      "args": null,
      "kind": "FragmentSpread",
      "name": "ForceCancelRunAlertFragment_run"
    },
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
        (v1/*: any*/),
        {
          "args": null,
          "kind": "FragmentSpread",
          "name": "RunTaskStageStatusPanelFragment_taskStage"
        },
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
                {
                  "args": null,
                  "kind": "FragmentSpread",
                  "name": "NoRunnerAlertFragment_job"
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
                    {
                      "alias": null,
                      "args": null,
                      "kind": "ScalarField",
                      "name": "name",
                      "storageKey": null
                    }
                  ],
                  "storageKey": null
                },
                {
                  "args": null,
                  "kind": "FragmentSpread",
                  "name": "RunTaskStagePolicyCheckPolicyCardFragment_gate"
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
                (v0/*: any*/),
                (v1/*: any*/),
                {
                  "args": null,
                  "kind": "FragmentSpread",
                  "name": "RunTaskStagePolicyCheckPolicyCardFragment_policy"
                }
              ],
              "storageKey": null
            },
            {
              "args": null,
              "kind": "FragmentSpread",
              "name": "RunTaskStagePolicyCheckPanelFragment_check"
            }
          ],
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Run",
  "abstractKey": null
};
})();

(node as any).hash = "06a0b1ab098cf572bd5c9af125f116ac";

export default node;
