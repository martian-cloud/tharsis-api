/**
 * @generated SignedSource<<723ee33b456d978dacc5fa7f313fc346>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type CheckResultStatus = "error" | "fail" | "pass" | "unknown" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type StateVersionCheckResultsFragment_checkResults$data = {
  readonly checkResults: ReadonlyArray<{
    readonly name: string;
    readonly status: CheckResultStatus;
    readonly " $fragmentSpreads": FragmentRefs<"StateVersionCheckResultRowFragment_checkResult">;
  }>;
  readonly " $fragmentType": "StateVersionCheckResultsFragment_checkResults";
};
export type StateVersionCheckResultsFragment_checkResults$key = {
  readonly " $data"?: StateVersionCheckResultsFragment_checkResults$data;
  readonly " $fragmentSpreads": FragmentRefs<"StateVersionCheckResultsFragment_checkResults">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "StateVersionCheckResultsFragment_checkResults",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "CheckResult",
      "kind": "LinkedField",
      "name": "checkResults",
      "plural": true,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "name",
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
          "args": null,
          "kind": "FragmentSpread",
          "name": "StateVersionCheckResultRowFragment_checkResult"
        }
      ],
      "storageKey": null
    }
  ],
  "type": "StateVersionInventory",
  "abstractKey": null
};

(node as any).hash = "70e16e0ec7b2f835c31e55cd5a4bb7ec";

export default node;
