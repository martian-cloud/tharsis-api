/**
 * @generated SignedSource<<6db9887c1e26a1a310b59de184952957>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type NamespaceOutputVisibilityLevel = "block_access" | "direct_group_and_subgroups" | "direct_group_only" | "global" | "root_group" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type OutputVisibilitySettingsFormFragment_outputVisibility$data = {
  readonly inherited: boolean;
  readonly namespacePath: string;
  readonly value: NamespaceOutputVisibilityLevel;
  readonly " $fragmentType": "OutputVisibilitySettingsFormFragment_outputVisibility";
};
export type OutputVisibilitySettingsFormFragment_outputVisibility$key = {
  readonly " $data"?: OutputVisibilitySettingsFormFragment_outputVisibility$data;
  readonly " $fragmentSpreads": FragmentRefs<"OutputVisibilitySettingsFormFragment_outputVisibility">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "OutputVisibilitySettingsFormFragment_outputVisibility",
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
    }
  ],
  "type": "NamespaceOutputVisibility",
  "abstractKey": null
};

(node as any).hash = "213efbfceabb758ac96b8dc643485d58";

export default node;
