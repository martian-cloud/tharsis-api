/**
 * @generated SignedSource<<a02cad33807ef6a5618353b8217207ff>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type NamespaceOutputVisibilityLevel = "block_access" | "direct_group_and_subgroups" | "direct_group_only" | "global" | "root_group" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type WorkspaceOutputVisibilitySettingsFragment_workspace$data = {
  readonly fullPath: string;
  readonly outputVisibility: {
    readonly inherited: boolean;
    readonly value: NamespaceOutputVisibilityLevel;
    readonly " $fragmentSpreads": FragmentRefs<"OutputVisibilitySettingsFormFragment_outputVisibility">;
  };
  readonly " $fragmentType": "WorkspaceOutputVisibilitySettingsFragment_workspace";
};
export type WorkspaceOutputVisibilitySettingsFragment_workspace$key = {
  readonly " $data"?: WorkspaceOutputVisibilitySettingsFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceOutputVisibilitySettingsFragment_workspace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "WorkspaceOutputVisibilitySettingsFragment_workspace",
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
  "type": "Workspace",
  "abstractKey": null
};

(node as any).hash = "4f96bfc7324172061cbbafa118ccb0aa";

export default node;
