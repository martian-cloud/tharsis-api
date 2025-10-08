/**
 * @generated SignedSource<<fd6dafcdfc31f9ad437ac000d4f207eb>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type StateVersionOutputListItemFragment_output$data = {
  readonly name: string;
  readonly sensitive: boolean;
  readonly type: string;
  readonly value: string;
  readonly " $fragmentType": "StateVersionOutputListItemFragment_output";
};
export type StateVersionOutputListItemFragment_output$key = {
  readonly " $data"?: StateVersionOutputListItemFragment_output$data;
  readonly " $fragmentSpreads": FragmentRefs<"StateVersionOutputListItemFragment_output">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "StateVersionOutputListItemFragment_output",
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
      "name": "value",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "type",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "sensitive",
      "storageKey": null
    }
  ],
  "type": "StateVersionOutput",
  "abstractKey": null
};

(node as any).hash = "0b2363fd7180e2e3b32f6a969ff5e9f7";

export default node;
