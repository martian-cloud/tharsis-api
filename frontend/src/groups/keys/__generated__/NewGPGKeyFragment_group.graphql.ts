/**
 * @generated SignedSource<<c13ea0428864871a8b09ee6912ca6647>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type NewGPGKeyFragment_group$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "NewGPGKeyFragment_group";
};
export type NewGPGKeyFragment_group$key = {
  readonly " $data"?: NewGPGKeyFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"NewGPGKeyFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "NewGPGKeyFragment_group",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "id",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "fullPath",
      "storageKey": null
    }
  ],
  "type": "Group",
  "abstractKey": null
};

(node as any).hash = "6ea02a6599750c7dee43d6c631cc7848";

export default node;
