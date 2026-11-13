/**
 * @generated SignedSource<<e93be4b674a4728a132c5141c6284e8c>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ManagedIdentityDetailsFragment_group$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "ManagedIdentityDetailsFragment_group";
};
export type ManagedIdentityDetailsFragment_group$key = {
  readonly " $data"?: ManagedIdentityDetailsFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"ManagedIdentityDetailsFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ManagedIdentityDetailsFragment_group",
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

(node as any).hash = "45c94d87c7b0b1a17dbd0441ddd98ec9";

export default node;
