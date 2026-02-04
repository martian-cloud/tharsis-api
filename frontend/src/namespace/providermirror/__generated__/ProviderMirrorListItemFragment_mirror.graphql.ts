/**
 * @generated SignedSource<<f145464389e9d2c84c05ffa29842b5c2>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ProviderMirrorListItemFragment_mirror$data = {
  readonly createdBy: string;
  readonly id: string;
  readonly metadata: {
    readonly createdAt: any;
  };
  readonly providerAddress: string;
  readonly version: string;
  readonly " $fragmentType": "ProviderMirrorListItemFragment_mirror";
};
export type ProviderMirrorListItemFragment_mirror$key = {
  readonly " $data"?: ProviderMirrorListItemFragment_mirror$data;
  readonly " $fragmentSpreads": FragmentRefs<"ProviderMirrorListItemFragment_mirror">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ProviderMirrorListItemFragment_mirror",
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
      "concreteType": "ResourceMetadata",
      "kind": "LinkedField",
      "name": "metadata",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "createdAt",
          "storageKey": null
        }
      ],
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "version",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "createdBy",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "providerAddress",
      "storageKey": null
    }
  ],
  "type": "TerraformProviderVersionMirror",
  "abstractKey": null
};

(node as any).hash = "04794ee9e2e01d26910f15669ad2f8d6";

export default node;
