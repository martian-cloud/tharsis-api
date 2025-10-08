/**
 * @generated SignedSource<<a3768efc1a4f7c6a3be84a0c44aced5e>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type VCSProviderListFragment_group$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "VCSProviderListFragment_group";
};
export type VCSProviderListFragment_group$key = {
  readonly " $data"?: VCSProviderListFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"VCSProviderListFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "VCSProviderListFragment_group",
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

(node as any).hash = "29c6e82e89b026a1c950c9d4d76eb2a7";

export default node;
