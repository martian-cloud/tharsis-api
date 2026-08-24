/**
 * @generated SignedSource<<eff8dabfe420844b26697be503ef3ecd>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupEditPackageFragment_group$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "GroupEditPackageFragment_group";
};
export type GroupEditPackageFragment_group$key = {
  readonly " $data"?: GroupEditPackageFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"GroupEditPackageFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "GroupEditPackageFragment_group",
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

(node as any).hash = "96e562766979eec102e8392498892a81";

export default node;
