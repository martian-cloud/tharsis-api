/**
 * @generated SignedSource<<daad10444eddac80a3edb00cbcca61cc>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupPackagesFragment_group$data = {
  readonly " $fragmentSpreads": FragmentRefs<"GroupEditPackageFragment_group" | "GroupNewPackageFragment_group" | "GroupPackageListFragment_group">;
  readonly " $fragmentType": "GroupPackagesFragment_group";
};
export type GroupPackagesFragment_group$key = {
  readonly " $data"?: GroupPackagesFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"GroupPackagesFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "GroupPackagesFragment_group",
  "selections": [
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "GroupPackageListFragment_group"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "GroupNewPackageFragment_group"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "GroupEditPackageFragment_group"
    }
  ],
  "type": "Group",
  "abstractKey": null
};

(node as any).hash = "0051c0a54ec57128a0aafb8f46b01e86";

export default node;
