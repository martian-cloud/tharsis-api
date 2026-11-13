/**
 * @generated SignedSource<<1ad7352787d33be46d8847dc897152d0>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type RunTaskStagePolicyCheckPolicyCardFragment_gate$data = {
  readonly " $fragmentSpreads": FragmentRefs<"RunTaskStagePolicyApproversBoxFragment_gate">;
  readonly " $fragmentType": "RunTaskStagePolicyCheckPolicyCardFragment_gate";
};
export type RunTaskStagePolicyCheckPolicyCardFragment_gate$key = {
  readonly " $data"?: RunTaskStagePolicyCheckPolicyCardFragment_gate$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunTaskStagePolicyCheckPolicyCardFragment_gate">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "RunTaskStagePolicyCheckPolicyCardFragment_gate",
  "selections": [
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "RunTaskStagePolicyApproversBoxFragment_gate"
    }
  ],
  "type": "RunGate",
  "abstractKey": null
};

(node as any).hash = "49c643188289623f8572da12898ba6fd";

export default node;
