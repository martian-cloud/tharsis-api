/**
 * @generated SignedSource<<e1ca1f04bd3395b5266a79c5938a2f5f>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type TerraformCheckResultStatus = "ERROR" | "FAIL" | "PASS" | "UNKNOWN" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type StateVersionCheckResultsFragment_checkResults$data = {
  readonly checkResults: ReadonlyArray<{
    readonly name: string;
    readonly status: TerraformCheckResultStatus;
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
      "concreteType": "TerraformCheckResult",
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
