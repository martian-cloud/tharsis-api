/**
 * @generated SignedSource<<c94303f5dbd90fa068832cd15c0bf441>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupAdvancedSettingsDeleteDialogFragment_group$data = {
  readonly fullPath: string;
  readonly name: string;
  readonly " $fragmentType": "GroupAdvancedSettingsDeleteDialogFragment_group";
};
export type GroupAdvancedSettingsDeleteDialogFragment_group$key = {
  readonly " $data"?: GroupAdvancedSettingsDeleteDialogFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"GroupAdvancedSettingsDeleteDialogFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "GroupAdvancedSettingsDeleteDialogFragment_group",
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
    }
  ],
  "type": "Group",
  "abstractKey": null
};

(node as any).hash = "8c2fd0608a0a5dd1cbcecb8fb37508dc";

export default node;
