/**
 * @generated SignedSource<<368d1d321afdb1a976e3a1297c2c7a70>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type WorkspaceRunnerSettingsFragment_workspace$data = {
  readonly fullPath: string;
  readonly runnerTags: {
    readonly inherited: boolean;
    readonly namespacePath: string;
    readonly value: ReadonlyArray<string>;
    readonly " $fragmentSpreads": FragmentRefs<"RunnerSettingsForm_runnerTags">;
  };
  readonly " $fragmentType": "WorkspaceRunnerSettingsFragment_workspace";
};
export type WorkspaceRunnerSettingsFragment_workspace$key = {
  readonly " $data"?: WorkspaceRunnerSettingsFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceRunnerSettingsFragment_workspace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "WorkspaceRunnerSettingsFragment_workspace",
  "selections": [
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
      "concreteType": "NamespaceRunnerTags",
      "kind": "LinkedField",
      "name": "runnerTags",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "inherited",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "namespacePath",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "value",
          "storageKey": null
        },
        {
          "args": null,
          "kind": "FragmentSpread",
          "name": "RunnerSettingsForm_runnerTags"
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Workspace",
  "abstractKey": null
};

(node as any).hash = "ee76124b14b50207d3e7d54490081423";

export default node;
