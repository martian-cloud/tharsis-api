/**
 * @generated SignedSource<<278408e1afc8c2ebdfb029af66932f96>>
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
export type RunDetailsApplyStageFragment_apply$data = {
  readonly apply: {
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
    readonly errorMessage: string | null | undefined;
    readonly jobs: {
      readonly totalCount: number;
    };
    readonly metadata: {
      readonly createdAt: any;
    };
    readonly status: ApplyStatus;
    readonly triggeredBy: string | null | undefined;
  } | null | undefined;
  readonly id: string;
  readonly plan: {
    readonly status: PlanStatus;
    readonly " $fragmentSpreads": FragmentRefs<"RunDetailsPlanSummaryFragment_plan">;
  };
  readonly status: RunStatus;
  readonly " $fragmentSpreads": FragmentRefs<"ForceCancelRunAlertFragment_run" | "RunVariablesFragment_variables">;
  readonly " $fragmentType": "RunDetailsApplyStageFragment_apply";
};
export type RunDetailsApplyStageFragment_apply$key = {
  readonly " $data"?: RunDetailsApplyStageFragment_apply$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunDetailsApplyStageFragment_apply">;
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
  "name": "RunDetailsApplyStageFragment_apply",
  "selections": [
    (v0/*: any*/),
    (v1/*: any*/),
    {
      "alias": null,
      "args": null,
      "concreteType": "Plan",
      "kind": "LinkedField",
      "name": "plan",
      "plural": false,
      "selections": [
        (v1/*: any*/),
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
        }
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

(node as any).hash = "265e9f4b6aea922a7c26bd108c0b58e4";

export default node;
