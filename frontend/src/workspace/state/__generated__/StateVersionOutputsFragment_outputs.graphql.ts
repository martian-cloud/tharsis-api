/**
 * @generated SignedSource<<5883cb4b4e3a47025a448e88b0c87b38>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type StateVersionOutputsFragment_outputs$data = {
  readonly outputs: ReadonlyArray<{
    readonly name: string;
    readonly " $fragmentSpreads": FragmentRefs<"StateVersionOutputListItemFragment_output">;
  }>;
  readonly " $fragmentType": "StateVersionOutputsFragment_outputs";
};
export type StateVersionOutputsFragment_outputs$key = {
  readonly " $data"?: StateVersionOutputsFragment_outputs$data;
  readonly " $fragmentSpreads": FragmentRefs<"StateVersionOutputsFragment_outputs">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "StateVersionOutputsFragment_outputs",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "StateVersionOutput",
      "kind": "LinkedField",
      "name": "outputs",
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
          "args": null,
          "kind": "FragmentSpread",
          "name": "StateVersionOutputListItemFragment_output"
        }
      ],
      "storageKey": null
    }
  ],
  "type": "StateVersion",
  "abstractKey": null
};

(node as any).hash = "1699456327cdc1c03177fa4d124dc2f5";

export default node;
