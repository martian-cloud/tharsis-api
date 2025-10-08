/**
 * @generated SignedSource<<7e949522793f1cf6fe8223c8d86ebc7a>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type FederatedRegistryDetailsFragment_group$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "FederatedRegistryDetailsFragment_group";
};
export type FederatedRegistryDetailsFragment_group$key = {
  readonly " $data"?: FederatedRegistryDetailsFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"FederatedRegistryDetailsFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "FederatedRegistryDetailsFragment_group",
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

(node as any).hash = "117ec2d413de4bc5a9437ab75e22ad68";

export default node;
