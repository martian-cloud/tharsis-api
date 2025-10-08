/**
 * @generated SignedSource<<a38e77f65cd2d9d31ada499a450a834c>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupSettingsFragment_group$data = {
  readonly fullPath: string;
  readonly " $fragmentSpreads": FragmentRefs<"GroupAdvancedSettingsFragment_group" | "GroupDriftDetectionSettingsFragment_group" | "GroupGeneralSettingsFragment_group" | "GroupRunnerSettingsFragment_group">;
  readonly " $fragmentType": "GroupSettingsFragment_group";
};
export type GroupSettingsFragment_group$key = {
  readonly " $data"?: GroupSettingsFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"GroupSettingsFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "GroupSettingsFragment_group",
  "selections": [
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
      "name": "GroupGeneralSettingsFragment_group"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "GroupAdvancedSettingsFragment_group"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "GroupRunnerSettingsFragment_group"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "GroupDriftDetectionSettingsFragment_group"
    }
  ],
  "type": "Group",
  "abstractKey": null
};

(node as any).hash = "9afefeb2c6948af6fc2473e956efb07f";

export default node;
