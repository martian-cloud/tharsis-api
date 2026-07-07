/**
 * @generated SignedSource<<ffd9e1aa256649ae0feec8177fc159e9>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type RunStatus = "applied" | "apply_queued" | "applying" | "canceled" | "discarded" | "errored" | "pending" | "plan_queued" | "planned" | "planned_and_finished" | "planning" | "queuing" | "queuing_apply" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type WorkspaceDetailsDriftDetectionFragment_workspace$data = {
  readonly assessment: {
    readonly completedAt: any | null | undefined;
    readonly hasDrift: boolean;
    readonly run: {
      readonly id: string;
      readonly status: RunStatus;
    } | null | undefined;
    readonly startedAt: any;
  } | null | undefined;
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "WorkspaceDetailsDriftDetectionFragment_workspace";
};
export type WorkspaceDetailsDriftDetectionFragment_workspace$key = {
  readonly " $data"?: WorkspaceDetailsDriftDetectionFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceDetailsDriftDetectionFragment_workspace">;
};

const node: ReaderFragment = (function(){
var v0 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
};
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "WorkspaceDetailsDriftDetectionFragment_workspace",
  "selections": [
    (v0/*: any*/),
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "fullPath",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "WorkspaceAssessment",
      "kind": "LinkedField",
      "name": "assessment",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "hasDrift",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "startedAt",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "completedAt",
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
            (v0/*: any*/),
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "status",
              "storageKey": null
            }
          ],
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Workspace",
  "abstractKey": null
};
})();

(node as any).hash = "9814940926704f320abe44420c34c410";

export default node;
