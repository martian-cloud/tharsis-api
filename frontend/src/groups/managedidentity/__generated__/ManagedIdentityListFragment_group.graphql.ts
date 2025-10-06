/**
 * @generated SignedSource<<68e539b819d4902024752f378c7da590>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ManagedIdentityListFragment_group$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "ManagedIdentityListFragment_group";
};
export type ManagedIdentityListFragment_group$key = {
  readonly " $data"?: ManagedIdentityListFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"ManagedIdentityListFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ManagedIdentityListFragment_group",
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

(node as any).hash = "68a3cf85ebaee51cf863af9161d073f0";

export default node;
