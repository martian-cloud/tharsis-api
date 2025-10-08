/**
 * @generated SignedSource<<21c0290a8acc99bc39580a300de31c63>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type StateVersionsFragment_stateVersions$data = {
  readonly fullPath: string;
  readonly " $fragmentSpreads": FragmentRefs<"StateVersionDetailsFragment_details" | "StateVersionListFragment_workspace">;
  readonly " $fragmentType": "StateVersionsFragment_stateVersions";
};
export type StateVersionsFragment_stateVersions$key = {
  readonly " $data"?: StateVersionsFragment_stateVersions$data;
  readonly " $fragmentSpreads": FragmentRefs<"StateVersionsFragment_stateVersions">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "StateVersionsFragment_stateVersions",
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
      "name": "StateVersionListFragment_workspace"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "StateVersionDetailsFragment_details"
    }
  ],
  "type": "Workspace",
  "abstractKey": null
};

(node as any).hash = "e7467536f0b19698df1623d8f1985788";

export default node;
