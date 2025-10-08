/**
 * @generated SignedSource<<5e370209dd1d5d1b56b6c4cd861c74a6>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type WorkspaceGeneralSettingsFragment_workspace$data = {
  readonly description: string;
  readonly fullPath: string;
  readonly name: string;
  readonly " $fragmentType": "WorkspaceGeneralSettingsFragment_workspace";
};
export type WorkspaceGeneralSettingsFragment_workspace$key = {
  readonly " $data"?: WorkspaceGeneralSettingsFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceGeneralSettingsFragment_workspace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "WorkspaceGeneralSettingsFragment_workspace",
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
      "name": "description",
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

(node as any).hash = "1b2dafabb9edec46ec9197785ae7491e";

export default node;
