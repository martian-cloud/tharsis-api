/**
 * @generated SignedSource<<ccd4b6bff1ce85cd40bbaa3b533c7742>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupPackageListFragment_group$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "GroupPackageListFragment_group";
};
export type GroupPackageListFragment_group$key = {
  readonly " $data"?: GroupPackageListFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"GroupPackageListFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "GroupPackageListFragment_group",
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

(node as any).hash = "ef236a0d91b3607b5ad9e86de102578a";

export default node;
