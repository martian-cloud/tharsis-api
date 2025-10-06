/**
 * @generated SignedSource<<d760d1f067ddf9c9d7408c1cb5086652>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type WorkspaceAdvancedSettingsFragment_workspace$data = {
  readonly fullPath: string;
  readonly name: string;
  readonly " $fragmentSpreads": FragmentRefs<"MigrateWorkspaceDialogFragment_workspace" | "WorkspaceAdvancedSettingsDeleteDialogFragment_workspace">;
  readonly " $fragmentType": "WorkspaceAdvancedSettingsFragment_workspace";
};
export type WorkspaceAdvancedSettingsFragment_workspace$key = {
  readonly " $data"?: WorkspaceAdvancedSettingsFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceAdvancedSettingsFragment_workspace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "WorkspaceAdvancedSettingsFragment_workspace",
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
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "WorkspaceAdvancedSettingsDeleteDialogFragment_workspace"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "MigrateWorkspaceDialogFragment_workspace"
    }
  ],
  "type": "Workspace",
  "abstractKey": null
};

(node as any).hash = "043de061c4ea607ad7f59a8116c890d6";

export default node;
