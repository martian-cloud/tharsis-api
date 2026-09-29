/**
 * @generated SignedSource<<857927ad3314772fb9769323879252a7>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type WorkspaceSettingsContentFragment_workspace$data = {
  readonly description: string;
  readonly fullPath: string;
  readonly name: string;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceAdvancedSettingsFragment_workspace" | "WorkspaceDriftDetectionSettingsFragment_workspace" | "WorkspaceGeneralSettingsFragment_workspace" | "WorkspaceLabelSettingsFragment_workspace" | "WorkspaceOutputVisibilitySettingsFragment_workspace" | "WorkspaceProviderMirrorSettingsFragment_workspace" | "WorkspaceRunSettingsFragment_workspace" | "WorkspaceRunnerSettingsFragment_workspace" | "WorkspaceStateSettingsFragment_workspace" | "WorkspaceVCSProviderSettingsFragment_workspace">;
  readonly " $fragmentType": "WorkspaceSettingsContentFragment_workspace";
};
export type WorkspaceSettingsContentFragment_workspace$key = {
  readonly " $data"?: WorkspaceSettingsContentFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceSettingsContentFragment_workspace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "WorkspaceSettingsContentFragment_workspace",
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
      "name": "WorkspaceProviderMirrorSettingsFragment_workspace"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "WorkspaceOutputVisibilitySettingsFragment_workspace"
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
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "WorkspaceLabelSettingsFragment_workspace"
    }
  ],
  "type": "Workspace",
  "abstractKey": null
};

(node as any).hash = "62e5986f0eacec411d6e8f22548cbd19";

export default node;
