/**
 * @generated SignedSource<<2e9ff57d7933b0b80b8028fe03b0e76e>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type RunnerAvailabilityStatus = "ASSIGNED" | "AVAILABLE" | "INACTIVE" | "NONE" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type NoRunnerAlertFragment_job$data = {
  readonly runnerAvailabilityStatus: RunnerAvailabilityStatus;
  readonly workspace: {
    readonly fullPath: string;
  };
  readonly " $fragmentType": "NoRunnerAlertFragment_job";
};
export type NoRunnerAlertFragment_job$key = {
  readonly " $data"?: NoRunnerAlertFragment_job$data;
  readonly " $fragmentSpreads": FragmentRefs<"NoRunnerAlertFragment_job">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "NoRunnerAlertFragment_job",
  "selections": [
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
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "fullPath",
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Job",
  "abstractKey": null
};

(node as any).hash = "d5a43d62c11607d284d643eef9532ef1";

export default node;
