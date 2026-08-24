/**
 * @generated SignedSource<<a657819a1da50ee10739b325d9d2f64a>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type ApplyStatus = "canceled" | "created" | "errored" | "finished" | "pending" | "queued" | "running" | "skipped" | "%future added value";
export type RunStatus = "applied" | "apply_queued" | "apply_queuing" | "applying" | "canceled" | "discarded" | "errored" | "pending" | "plan_queued" | "plan_queuing" | "planned" | "planned_and_finished" | "planning" | "post_plan_awaiting_decision" | "post_plan_completed" | "post_plan_running" | "pre_apply_awaiting_decision" | "pre_apply_completed" | "pre_apply_queuing" | "pre_apply_running" | "pre_plan_awaiting_decision" | "pre_plan_completed" | "pre_plan_queuing" | "pre_plan_running" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type RunListItemFragment_run$data = {
  readonly apply: {
    readonly status: ApplyStatus;
  } | null | undefined;
  readonly assessment: boolean;
  readonly createdBy: string;
  readonly hasAdvisoryFailures: boolean;
  readonly id: string;
  readonly isDestroy: boolean;
  readonly metadata: {
    readonly createdAt: any;
    readonly trn: string;
  };
  readonly status: RunStatus;
  readonly workspace: {
    readonly fullPath: string;
  };
  readonly " $fragmentSpreads": FragmentRefs<"RunStageIconsFragment_run">;
  readonly " $fragmentType": "RunListItemFragment_run";
};
export type RunListItemFragment_run$key = {
  readonly " $data"?: RunListItemFragment_run$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunListItemFragment_run">;
};

const node: ReaderFragment = (function(){
var v0 = {
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
  "name": "RunListItemFragment_run",
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
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "id",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "createdBy",
      "storageKey": null
    },
    (v0/*: any*/),
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
      "name": "hasAdvisoryFailures",
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
          "name": "fullPath",
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
        (v0/*: any*/)
      ],
      "storageKey": null
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "RunStageIconsFragment_run"
    }
  ],
  "type": "Run",
  "abstractKey": null
};
})();

(node as any).hash = "db65a6eb2a49fa6d719baead76dd155a";

export default node;
