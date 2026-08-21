/**
 * @generated SignedSource<<ccdc3c5caf7a18b27ec918a5ea9fc127>>
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
export type RunTaskStageStatusPanelFragment_taskStage$data = {
  readonly policyChecks: ReadonlyArray<{
    readonly currentJob: {
      readonly cancelRequested: boolean;
    } | null | undefined;
  }>;
  readonly stageName: RunTaskStageName;
  readonly status: RunTaskStageStatus;
  readonly " $fragmentType": "RunTaskStageStatusPanelFragment_taskStage";
};
export type RunTaskStageStatusPanelFragment_taskStage$key = {
  readonly " $data"?: RunTaskStageStatusPanelFragment_taskStage$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunTaskStageStatusPanelFragment_taskStage">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "RunTaskStageStatusPanelFragment_taskStage",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "stageName",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "status",
      "storageKey": null
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
            }
          ],
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "RunTaskStage",
  "abstractKey": null
};

(node as any).hash = "03ccec52106d6fa150b14e8ad1dbac6f";

export default node;
