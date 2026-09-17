/**
 * @generated SignedSource<<776ad4bb6c851a4eee664a49b2b69272>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type ApplyStatus = "canceled" | "created" | "errored" | "finished" | "pending" | "queued" | "running" | "skipped" | "%future added value";
export type PlanStatus = "canceled" | "created" | "errored" | "finished" | "pending" | "queued" | "running" | "skipped" | "%future added value";
export type PolicyCheckStatus = "CANCELED" | "CREATED" | "ERRORED" | "OVERRIDDEN" | "PASSED" | "PENDING" | "QUEUED" | "RUNNING" | "SKIPPED" | "SOFT_FAILED" | "%future added value";
export type RunStatus = "applied" | "apply_queued" | "apply_queuing" | "applying" | "canceled" | "discarded" | "errored" | "pending" | "plan_queued" | "plan_queuing" | "planned" | "planned_and_finished" | "planning" | "post_apply_running" | "post_plan_awaiting_decision" | "post_plan_running" | "pre_apply_awaiting_decision" | "pre_apply_completed" | "pre_apply_queuing" | "pre_apply_running" | "pre_plan_awaiting_decision" | "pre_plan_completed" | "pre_plan_queuing" | "pre_plan_running" | "%future added value";
export type RunTaskStageName = "POST_APPLY" | "POST_PLAN" | "PRE_APPLY" | "PRE_PLAN" | "%future added value";
export type RunTaskStageStatus = "AWAITING_OVERRIDE" | "CANCELED" | "COMPLETED" | "CREATED" | "ERRORED" | "PENDING" | "RUNNING" | "SKIPPED" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type RunDetailsSidebarFragment_details$data = {
  readonly annotations: ReadonlyArray<{
    readonly key: string;
  }>;
  readonly apply: {
    readonly currentJob: {
      readonly cancelRequested: boolean;
      readonly runnerPath: string | null | undefined;
    } | null | undefined;
    readonly metadata: {
      readonly createdAt: any;
    };
    readonly status: ApplyStatus;
  } | null | undefined;
  readonly assessment: boolean;
  readonly autoApply: boolean;
  readonly configurationVersion: {
    readonly id: string;
  } | null | undefined;
  readonly createdBy: string;
  readonly hasAdvisoryFailures: boolean;
  readonly id: string;
  readonly isDestroy: boolean;
  readonly metadata: {
    readonly createdAt: any;
    readonly trn: string;
  };
  readonly moduleSource: string | null | undefined;
  readonly moduleVersion: string | null | undefined;
  readonly plan: {
    readonly currentJob: {
      readonly cancelRequested: boolean;
      readonly runnerPath: string | null | undefined;
    } | null | undefined;
    readonly metadata: {
      readonly createdAt: any;
    };
    readonly status: PlanStatus;
  };
  readonly status: RunStatus;
  readonly taskStages: ReadonlyArray<{
    readonly policyChecks: ReadonlyArray<{
      readonly stageName: RunTaskStageName;
      readonly status: PolicyCheckStatus;
    }>;
    readonly stageName: RunTaskStageName;
    readonly status: RunTaskStageStatus;
  }>;
  readonly workspace: {
    readonly fullPath: string;
  };
  readonly " $fragmentSpreads": FragmentRefs<"RunAnnotationsFragment_run">;
  readonly " $fragmentType": "RunDetailsSidebarFragment_details";
};
export type RunDetailsSidebarFragment_details$key = {
  readonly " $data"?: RunDetailsSidebarFragment_details$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunDetailsSidebarFragment_details">;
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
},
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "createdAt",
  "storageKey": null
},
v3 = [
  (v1/*: any*/),
  {
    "alias": null,
    "args": null,
    "concreteType": "ResourceMetadata",
    "kind": "LinkedField",
    "name": "metadata",
    "plural": false,
    "selections": [
      (v2/*: any*/)
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
      }
    ],
    "storageKey": null
  }
],
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "stageName",
  "storageKey": null
};
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "RunDetailsSidebarFragment_details",
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
      "name": "autoApply",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "hasAdvisoryFailures",
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
      "concreteType": "RunAnnotation",
      "kind": "LinkedField",
      "name": "annotations",
      "plural": true,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "key",
          "storageKey": null
        }
      ],
      "storageKey": null
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "RunAnnotationsFragment_run"
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
          "name": "fullPath",
          "storageKey": null
        }
      ],
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
        (v2/*: any*/),
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
    {
      "alias": null,
      "args": null,
      "concreteType": "ConfigurationVersion",
      "kind": "LinkedField",
      "name": "configurationVersion",
      "plural": false,
      "selections": [
        (v0/*: any*/)
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
      "selections": (v3/*: any*/),
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
        (v4/*: any*/),
        (v1/*: any*/),
        {
          "alias": null,
          "args": null,
          "concreteType": "PolicyCheck",
          "kind": "LinkedField",
          "name": "policyChecks",
          "plural": true,
          "selections": [
            (v1/*: any*/),
            (v4/*: any*/)
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
      "selections": (v3/*: any*/),
      "storageKey": null
    }
  ],
  "type": "Run",
  "abstractKey": null
};
})();

(node as any).hash = "b40bca310f52d35006484557d96452ee";

export default node;
