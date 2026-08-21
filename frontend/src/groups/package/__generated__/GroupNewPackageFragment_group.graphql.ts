/**
 * @generated SignedSource<<2fd68610ff06a946149ed6f2f8413e6e>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupNewPackageFragment_group$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "GroupNewPackageFragment_group";
};
export type GroupNewPackageFragment_group$key = {
  readonly " $data"?: GroupNewPackageFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"GroupNewPackageFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "GroupNewPackageFragment_group",
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

(node as any).hash = "edbccbd2475577384dedb853224522e6";

export default node;
