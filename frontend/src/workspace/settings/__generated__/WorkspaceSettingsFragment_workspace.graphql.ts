/**
 * @generated SignedSource<<bd4ce80283a49e1664ff8243689fb9af>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type WorkspaceSettingsFragment_workspace$data = {
  readonly description: string;
  readonly fullPath: string;
  readonly name: string;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceAdvancedSettingsFragment_workspace" | "WorkspaceDriftDetectionSettingsFragment_workspace" | "WorkspaceGeneralSettingsFragment_workspace" | "WorkspaceRunSettingsFragment_workspace" | "WorkspaceRunnerSettingsFragment_workspace" | "WorkspaceStateSettingsFragment_workspace" | "WorkspaceVCSProviderSettingsFragment_workspace">;
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
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "WorkspaceGeneralSettingsFragment_workspace"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "WorkspaceRunnerSettingsFragment_workspace"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "WorkspaceRunSettingsFragment_workspace"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "WorkspaceDriftDetectionSettingsFragment_workspace"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "WorkspaceAdvancedSettingsFragment_workspace"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "WorkspaceVCSProviderSettingsFragment_workspace"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "WorkspaceStateSettingsFragment_workspace"
    }
  ],
  "type": "Workspace",
  "abstractKey": null
};

(node as any).hash = "43765be7d5d5a72c31d450e3c35726b8";

export default node;
