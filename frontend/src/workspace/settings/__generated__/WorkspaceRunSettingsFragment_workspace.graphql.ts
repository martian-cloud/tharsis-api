/**
 * @generated SignedSource<<7ae7948984a4fda510daca16ef6f2e18>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type WorkspaceRunSettingsFragment_workspace$data = {
  readonly description: string;
  readonly fullPath: string;
  readonly maxJobDuration: number;
  readonly name: string;
  readonly preventDestroyPlan: boolean;
  readonly terraformVersion: string;
  readonly " $fragmentSpreads": FragmentRefs<"MaxJobDurationSettingFragment_workspace">;
  readonly " $fragmentType": "WorkspaceRunSettingsFragment_workspace";
};
export type WorkspaceRunSettingsFragment_workspace$key = {
  readonly " $data"?: WorkspaceRunSettingsFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceRunSettingsFragment_workspace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "WorkspaceRunSettingsFragment_workspace",
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
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "maxJobDuration",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "terraformVersion",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "preventDestroyPlan",
      "storageKey": null
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "MaxJobDurationSettingFragment_workspace"
    }
  ],
  "type": "Workspace",
  "abstractKey": null
};

(node as any).hash = "b6ff6c64a0d5f9d54dfa37673faacf6d";

export default node;
