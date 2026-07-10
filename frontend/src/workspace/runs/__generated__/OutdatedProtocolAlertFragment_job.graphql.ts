/**
 * @generated SignedSource<<574b78589b640fa6d682abfb7ea5a7fe>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type OutdatedProtocolAlertFragment_job$data = {
  readonly outdatedJobProtocolVersion: boolean;
  readonly " $fragmentType": "OutdatedProtocolAlertFragment_job";
};
export type OutdatedProtocolAlertFragment_job$key = {
  readonly " $data"?: OutdatedProtocolAlertFragment_job$data;
  readonly " $fragmentSpreads": FragmentRefs<"OutdatedProtocolAlertFragment_job">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "OutdatedProtocolAlertFragment_job",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "outdatedJobProtocolVersion",
      "storageKey": null
    }
  ],
  "type": "Job",
  "abstractKey": null
};

(node as any).hash = "98150175298902c8a6522474027369aa";

export default node;
