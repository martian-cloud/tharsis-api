/**
 * @generated SignedSource<<18bc9cdf7d9777165214c3529059cbd5>>
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
  readonly " $fragmentSpreads": FragmentRefs<"GroupAdvancedSettingsFragment_group" | "GroupDriftDetectionSettingsFragment_group" | "GroupGeneralSettingsFragment_group" | "GroupProviderMirrorSettingsFragment_group" | "GroupRunnerSettingsFragment_group">;
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
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "GroupProviderMirrorSettingsFragment_group"
    }
  ],
  "type": "Group",
  "abstractKey": null
};

(node as any).hash = "1f39da762584e55170e189aa569e77f8";

export default node;
