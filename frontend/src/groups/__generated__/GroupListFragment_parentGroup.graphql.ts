/**
 * @generated SignedSource<<4b4a359d00fe346bff53355cf0da6a3f>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupListFragment_parentGroup$data = {
  readonly id: string;
  readonly " $fragmentType": "GroupListFragment_parentGroup";
};
export type GroupListFragment_parentGroup$key = {
  readonly " $data"?: GroupListFragment_parentGroup$data;
  readonly " $fragmentSpreads": FragmentRefs<"GroupListFragment_parentGroup">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "GroupListFragment_parentGroup",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "id",
      "storageKey": null
    }
  ],
  "type": "Group",
  "abstractKey": null
};

(node as any).hash = "2d695edcde16b2076bda95edb575f2a2";

export default node;
