/**
 * @generated SignedSource<<4ca8fce143f8332390a1dbb9a41dc963>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type NamespaceOutputVisibilityLevel = "block_access" | "direct_group_and_subgroups" | "direct_group_only" | "global" | "root_group" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type GroupOutputVisibilitySettingsFragment_group$data = {
  readonly fullPath: string;
  readonly outputVisibility: {
    readonly inherited: boolean;
    readonly value: NamespaceOutputVisibilityLevel;
    readonly " $fragmentSpreads": FragmentRefs<"OutputVisibilitySettingsFormFragment_outputVisibility">;
  };
  readonly " $fragmentType": "GroupOutputVisibilitySettingsFragment_group";
};
export type GroupOutputVisibilitySettingsFragment_group$key = {
  readonly " $data"?: GroupOutputVisibilitySettingsFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"GroupOutputVisibilitySettingsFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "GroupOutputVisibilitySettingsFragment_group",
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
      "concreteType": "NamespaceOutputVisibility",
      "kind": "LinkedField",
      "name": "outputVisibility",
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
          "name": "value",
          "storageKey": null
        },
        {
          "args": null,
          "kind": "FragmentSpread",
          "name": "OutputVisibilitySettingsFormFragment_outputVisibility"
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Group",
  "abstractKey": null
};

(node as any).hash = "783c46724ef0b00b7e0d7b7594778d7c";

export default node;
