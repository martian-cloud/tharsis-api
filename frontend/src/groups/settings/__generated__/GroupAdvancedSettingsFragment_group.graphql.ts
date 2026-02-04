/**
 * @generated SignedSource<<440c08c25773b0e27e822ae00eb615d9>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupAdvancedSettingsFragment_group$data = {
  readonly fullPath: string;
  readonly name: string;
  readonly " $fragmentSpreads": FragmentRefs<"MigrateGroupDialogFragment_group">;
  readonly " $fragmentType": "GroupAdvancedSettingsFragment_group";
};
export type GroupAdvancedSettingsFragment_group$key = {
  readonly " $data"?: GroupAdvancedSettingsFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"GroupAdvancedSettingsFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "GroupAdvancedSettingsFragment_group",
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
      "name": "MigrateGroupDialogFragment_group"
    }
  ],
  "type": "Group",
  "abstractKey": null
};

(node as any).hash = "d0e6b91b371ba16edf08bf2c8ceec601";

export default node;
