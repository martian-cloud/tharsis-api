/**
 * @generated SignedSource<<5fa2c87975c3c41ba2f05847bd476787>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ForceCancelRunButtonFragment_run$data = {
  readonly id: string;
  readonly " $fragmentSpreads": FragmentRefs<"ForceCancelRunButtonDialogFragment_run">;
  readonly " $fragmentType": "ForceCancelRunButtonFragment_run";
};
export type ForceCancelRunButtonFragment_run$key = {
  readonly " $data"?: ForceCancelRunButtonFragment_run$data;
  readonly " $fragmentSpreads": FragmentRefs<"ForceCancelRunButtonFragment_run">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ForceCancelRunButtonFragment_run",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "id",
      "storageKey": null
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "ForceCancelRunButtonDialogFragment_run"
    }
  ],
  "type": "Run",
  "abstractKey": null
};

(node as any).hash = "ba4e608975407198b8ec72a2af736772";

export default node;
