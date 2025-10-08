/**
 * @generated SignedSource<<62a4382af33851aab77af0dceef5da03>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type ApplyStatus = "canceled" | "created" | "errored" | "finished" | "pending" | "queued" | "running" | "%future added value";
export type JobStatus = "finished" | "pending" | "queued" | "running" | "%future added value";
export type PlanStatus = "canceled" | "errored" | "finished" | "pending" | "queued" | "running" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type RunDetailsPlanStageFragment_plan$data = {
  readonly apply: {
    readonly status: ApplyStatus;
  } | null | undefined;
  readonly createdBy: string;
  readonly id: string;
  readonly plan: {
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
      readonly " $fragmentSpreads": FragmentRefs<"JobLogsFragment_logs" | "NoRunnerAlertFragment_job" | "RunJobDialog_currentJob">;
    } | null | undefined;
    readonly diffSize: number;
    readonly errorMessage: string | null | undefined;
    readonly hasChanges: boolean;
    readonly metadata: {
      readonly createdAt: any;
      readonly trn: string;
    };
    readonly status: PlanStatus;
    readonly " $fragmentSpreads": FragmentRefs<"RunDetailsPlanSummaryFragment_plan">;
  };
  readonly workspace: {
    readonly locked: boolean;
    readonly metadata: {
      readonly updatedAt: any;
    };
  };
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
      "concreteType": "Workspace",
      "kind": "LinkedField",
      "name": "workspace",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "locked",
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
            },
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
            },
            {
              "args": null,
              "kind": "FragmentSpread",
              "name": "JobLogsFragment_logs"
            },
            {
              "args": null,
              "kind": "FragmentSpread",
              "name": "RunJobDialog_currentJob"
            }
          ],
          "storageKey": null
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

(node as any).hash = "c079de4ee265f34b65f58e059d8555dc";

export default node;
