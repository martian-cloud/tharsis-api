/**
 * @generated SignedSource<<e32aa162d4f4c693a295c4a0f8427d13>>
 * @lightSyntaxTransform
 * @nogrep
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
