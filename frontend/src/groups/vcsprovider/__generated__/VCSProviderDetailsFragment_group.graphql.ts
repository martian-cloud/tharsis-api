/**
 * @generated SignedSource<<b9cdd2160ef6147078bab529bc15bb1a>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type VCSProviderDetailsFragment_group$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "VCSProviderDetailsFragment_group";
};
export type VCSProviderDetailsFragment_group$key = {
  readonly " $data"?: VCSProviderDetailsFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"VCSProviderDetailsFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "VCSProviderDetailsFragment_group",
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

(node as any).hash = "66f6b3eb1aa8d5a58a9ea470a649fa51";

export default node;
