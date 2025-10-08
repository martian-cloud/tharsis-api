/**
 * @generated SignedSource<<e33a0b0062060237258dce9093968c56>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type StateVersionFileFragment_stateVersion$data = {
  readonly id: string;
  readonly " $fragmentType": "StateVersionFileFragment_stateVersion";
};
export type StateVersionFileFragment_stateVersion$key = {
  readonly " $data"?: StateVersionFileFragment_stateVersion$data;
  readonly " $fragmentSpreads": FragmentRefs<"StateVersionFileFragment_stateVersion">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "StateVersionFileFragment_stateVersion",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "id",
      "storageKey": null
    }
  ],
  "type": "StateVersion",
  "abstractKey": null
};

(node as any).hash = "7d3afabeb51c28b79846d35b074dc075";

export default node;
