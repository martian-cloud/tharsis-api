/**
 * @generated SignedSource<<a6763f50e5dac94b0cc242789ed4c6fb>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type FederatedRegistryListItemFragment_federatedRegistry$data = {
  readonly group: {
    readonly fullPath: string;
  };
  readonly hostname: string;
  readonly id: string;
  readonly metadata: {
    readonly updatedAt: any;
  };
  readonly " $fragmentType": "FederatedRegistryListItemFragment_federatedRegistry";
};
export type FederatedRegistryListItemFragment_federatedRegistry$key = {
  readonly " $data"?: FederatedRegistryListItemFragment_federatedRegistry$data;
  readonly " $fragmentSpreads": FragmentRefs<"FederatedRegistryListItemFragment_federatedRegistry">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "FederatedRegistryListItemFragment_federatedRegistry",
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
      "name": "hostname",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "ResourceMetadata",
      "kind": "LinkedField",
      "name": "metadata",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "updatedAt",
          "storageKey": null
        }
      ],
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "Group",
      "kind": "LinkedField",
      "name": "group",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "fullPath",
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "FederatedRegistry",
  "abstractKey": null
};

(node as any).hash = "523f9a981fc57c28dd9b7078740b02e3";

export default node;
