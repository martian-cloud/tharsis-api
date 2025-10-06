/**
 * @generated SignedSource<<d6d73f64c377f688f31cbfbaf0c94e09>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GPGKeysFragment_group$data = {
  readonly " $fragmentSpreads": FragmentRefs<"GPGKeyListFragment_group" | "NewGPGKeyFragment_group">;
  readonly " $fragmentType": "GPGKeysFragment_group";
};
export type GPGKeysFragment_group$key = {
  readonly " $data"?: GPGKeysFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"GPGKeysFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "GPGKeysFragment_group",
  "selections": [
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "GPGKeyListFragment_group"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "NewGPGKeyFragment_group"
    }
  ],
  "type": "Group",
  "abstractKey": null
};

(node as any).hash = "897da5e6f217aa785188e43781ad9b8c";

export default node;
