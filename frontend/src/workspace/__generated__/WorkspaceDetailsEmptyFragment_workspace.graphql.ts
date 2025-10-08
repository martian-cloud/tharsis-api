/**
 * @generated SignedSource<<027891556db0b6e27959a37c6af8edac>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type WorkspaceDetailsEmptyFragment_workspace$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "WorkspaceDetailsEmptyFragment_workspace";
};
export type WorkspaceDetailsEmptyFragment_workspace$key = {
  readonly " $data"?: WorkspaceDetailsEmptyFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceDetailsEmptyFragment_workspace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "WorkspaceDetailsEmptyFragment_workspace",
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
  "type": "Workspace",
  "abstractKey": null
};

(node as any).hash = "fadf0961fc7985536a47bf5b357048e0";

export default node;
