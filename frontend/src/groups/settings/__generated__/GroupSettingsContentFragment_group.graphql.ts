/**
 * @generated SignedSource<<d9f63e766cb3f2e2348818e472c3bdb5>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupSettingsContentFragment_group$data = {
  readonly " $fragmentSpreads": FragmentRefs<"GroupAdvancedSettingsFragment_group" | "GroupDriftDetectionSettingsFragment_group" | "GroupGeneralSettingsFragment_group" | "GroupOutputVisibilitySettingsFragment_group" | "GroupProviderMirrorSettingsFragment_group" | "GroupRunnerSettingsFragment_group">;
  readonly " $fragmentType": "GroupSettingsContentFragment_group";
};
export type GroupSettingsContentFragment_group$key = {
  readonly " $data"?: GroupSettingsContentFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"GroupSettingsContentFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "GroupSettingsContentFragment_group",
  "selections": [
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
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "GroupOutputVisibilitySettingsFragment_group"
    }
  ],
  "type": "Group",
  "abstractKey": null
};

(node as any).hash = "6e7654fafae60f7a0784dddbd5e7bc8c";

export default node;
