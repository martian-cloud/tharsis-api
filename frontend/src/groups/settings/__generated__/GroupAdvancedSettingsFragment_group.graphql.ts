/**
 * @generated SignedSource<<6018e1b002455dff880140e0697b8782>>
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
  readonly " $fragmentSpreads": FragmentRefs<"GroupAdvancedSettingsDeleteDialogFragment_group" | "MigrateGroupDialogFragment_group">;
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
      "name": "GroupAdvancedSettingsDeleteDialogFragment_group"
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

(node as any).hash = "2e4a8d3f6d8e02810195b4fadbe3f5fe";

export default node;
