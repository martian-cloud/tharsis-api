/**
 * @generated SignedSource<<c64957a3ebede6bb887cde50cf6c7a4b>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ForceCancelRunAlertFragment_run$data = {
  readonly forceCancelAvailableAt: any | null | undefined;
  readonly " $fragmentSpreads": FragmentRefs<"ForceCancelRunButtonFragment_run">;
  readonly " $fragmentType": "ForceCancelRunAlertFragment_run";
};
export type ForceCancelRunAlertFragment_run$key = {
  readonly " $data"?: ForceCancelRunAlertFragment_run$data;
  readonly " $fragmentSpreads": FragmentRefs<"ForceCancelRunAlertFragment_run">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ForceCancelRunAlertFragment_run",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "forceCancelAvailableAt",
      "storageKey": null
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "ForceCancelRunButtonFragment_run"
    }
  ],
  "type": "Run",
  "abstractKey": null
};

(node as any).hash = "e77ef8cbec8faccd6c6e8d3cd213a035";

export default node;
