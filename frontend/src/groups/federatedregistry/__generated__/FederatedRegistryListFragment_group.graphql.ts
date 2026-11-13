/**
 * @generated SignedSource<<75aea84e9a4a9d18b764ddd639ad9358>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type FederatedRegistryListFragment_group$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "FederatedRegistryListFragment_group";
};
export type FederatedRegistryListFragment_group$key = {
  readonly " $data"?: FederatedRegistryListFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"FederatedRegistryListFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "FederatedRegistryListFragment_group",
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

(node as any).hash = "18ab470860554d173f8fe643dabc517a";

export default node;
