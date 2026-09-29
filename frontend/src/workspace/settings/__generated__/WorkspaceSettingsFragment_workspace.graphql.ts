/**
 * @generated SignedSource<<330e69d31084fdba79b729cb11ff5679>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type WorkspaceSettingsFragment_workspace$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "WorkspaceSettingsFragment_workspace";
};
export type WorkspaceSettingsFragment_workspace$key = {
  readonly " $data"?: WorkspaceSettingsFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceSettingsFragment_workspace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "WorkspaceSettingsFragment_workspace",
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

(node as any).hash = "e8636926ffda42f835d835ffbfc47b17";

export default node;
