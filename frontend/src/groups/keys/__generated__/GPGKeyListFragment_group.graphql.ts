/**
 * @generated SignedSource<<8f204e574793ccb04715b9f19d8380ba>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GPGKeyListFragment_group$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "GPGKeyListFragment_group";
};
export type GPGKeyListFragment_group$key = {
  readonly " $data"?: GPGKeyListFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"GPGKeyListFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "GPGKeyListFragment_group",
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

(node as any).hash = "57baae1a7ccb209de62792ed7d8b674a";

export default node;
