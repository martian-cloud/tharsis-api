/**
 * @generated SignedSource<<2426e733bb0a3b01396266c6b520eea3>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type HomeWorkspaceListItemFragment_workspace$data = {
  readonly fullPath: string;
  readonly name: string;
  readonly " $fragmentType": "HomeWorkspaceListItemFragment_workspace";
};
export type HomeWorkspaceListItemFragment_workspace$key = {
  readonly " $data"?: HomeWorkspaceListItemFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"HomeWorkspaceListItemFragment_workspace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "HomeWorkspaceListItemFragment_workspace",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "name",
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
  "type": "Workspace",
  "abstractKey": null
};

(node as any).hash = "b38c3aca15f8b5426970dbab1d3476a4";

export default node;
