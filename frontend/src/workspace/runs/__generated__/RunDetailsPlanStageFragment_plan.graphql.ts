/**
 * @generated SignedSource<<39431a934d33e6259a71a4171d287a39>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type ApplyStatus = "canceled" | "created" | "errored" | "finished" | "pending" | "queued" | "running" | "skipped" | "%future added value";
export type JobStatus = "canceled" | "canceling" | "failed" | "finished" | "pending" | "queued" | "running" | "%future added value";
export type PlanStatus = "canceled" | "created" | "errored" | "finished" | "pending" | "queued" | "running" | "%future added value";
export type RunStatus = "applied" | "apply_queued" | "applying" | "canceled" | "discarded" | "errored" | "pending" | "plan_queued" | "planned" | "planned_and_finished" | "planning" | "queuing" | "queuing_apply" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type RunDetailsPlanStageFragment_plan$data = {
  readonly apply: {
    readonly status: ApplyStatus;
  } | null | undefined;
  readonly createdBy: string;
  readonly id: string;
  readonly plan: {
    readonly checkResults: ReadonlyArray<{
      readonly " $fragmentSpreads": FragmentRefs<"CheckResultsPanelFragment_checkResult">;
    }>;
    readonly currentJob: {
      readonly cancelRequested: boolean;
      readonly id: string;
      readonly status: JobStatus;
      readonly timestamps: {
        readonly finishedAt: any | null | undefined;
        readonly pendingAt: any | null | undefined;
        readonly queuedAt: any | null | undefined;
        readonly runningAt: any | null | undefined;
      };
      readonly " $fragmentSpreads": FragmentRefs<"NoRunnerAlertFragment_job">;
    } | null | undefined;
    readonly diffSize: number;
    readonly errorMessage: string | null | undefined;
    readonly hasChanges: boolean;
    readonly jobs: {
      readonly totalCount: number;
    };
    readonly metadata: {
      readonly createdAt: any;
    };
    readonly status: PlanStatus;
    readonly " $fragmentSpreads": FragmentRefs<"RunDetailsPlanSummaryFragment_plan">;
  };
  readonly status: RunStatus;
  readonly " $fragmentSpreads": FragmentRefs<"ForceCancelRunAlertFragment_run" | "RunVariablesFragment_variables">;
  readonly " $fragmentType": "RunDetailsPlanStageFragment_plan";
};
export type RunDetailsPlanStageFragment_plan$key = {
  readonly " $data"?: RunDetailsPlanStageFragment_plan$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunDetailsPlanStageFragment_plan">;
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
  "name": "RunDetailsPlanStageFragment_plan",
  "selections": [
    (v0/*: any*/),
    (v1/*: any*/),
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
      "concreteType": "Plan",
      "kind": "LinkedField",
      "name": "plan",
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
        (v1/*: any*/),
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
          "concreteType": "CheckResult",
          "kind": "LinkedField",
          "name": "checkResults",
          "plural": true,
          "selections": [
            {
              "args": null,
              "kind": "FragmentSpread",
              "name": "CheckResultsPanelFragment_checkResult"
            }
          ],
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
            (v0/*: any*/),
            (v1/*: any*/),
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
              "args": null,
              "kind": "FragmentSpread",
              "name": "NoRunnerAlertFragment_job"
            }
          ],
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
          "args": null,
          "kind": "FragmentSpread",
          "name": "RunDetailsPlanSummaryFragment_plan"
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
        (v1/*: any*/)
      ],
      "storageKey": null
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "RunVariablesFragment_variables"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "ForceCancelRunAlertFragment_run"
    }
  ],
  "type": "Run",
  "abstractKey": null
};
})();

(node as any).hash = "4b3d797246a2121810678c9f709bd92b";

export default node;
