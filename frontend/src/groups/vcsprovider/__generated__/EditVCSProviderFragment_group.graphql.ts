/**
 * @generated SignedSource<<e9566721a5dff13f0114a74e8c907085>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type EditVCSProviderFragment_group$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "EditVCSProviderFragment_group";
};
export type EditVCSProviderFragment_group$key = {
  readonly " $data"?: EditVCSProviderFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"EditVCSProviderFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "EditVCSProviderFragment_group",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "id",
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

(node as any).hash = "5405589b715981f55904cfe17900265d";

export default node;
