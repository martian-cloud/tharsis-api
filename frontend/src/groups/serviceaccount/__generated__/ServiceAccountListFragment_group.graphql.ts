/**
 * @generated SignedSource<<53ac2ce0f9c91cbe738a4bf6582b152d>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ServiceAccountListFragment_group$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "ServiceAccountListFragment_group";
};
export type ServiceAccountListFragment_group$key = {
  readonly " $data"?: ServiceAccountListFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"ServiceAccountListFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ServiceAccountListFragment_group",
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

(node as any).hash = "0ceb188a9d5ba9be09af4f87f68d2b8c";

export default node;
