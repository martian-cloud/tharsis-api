/**
 * @generated SignedSource<<9ae70c8a409b5da8ec9fc27a98ed59a8>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TerraformProviderSearchListItemFragment_provider$data = {
  readonly id: string;
  readonly latestVersion: {
    readonly createdBy: string;
    readonly metadata: {
      readonly createdAt: any;
    };
    readonly version: string;
  } | null | undefined;
  readonly name: string;
  readonly private: boolean;
  readonly registryNamespace: string;
  readonly " $fragmentType": "TerraformProviderSearchListItemFragment_provider";
};
export type TerraformProviderSearchListItemFragment_provider$key = {
  readonly " $data"?: TerraformProviderSearchListItemFragment_provider$data;
  readonly " $fragmentSpreads": FragmentRefs<"TerraformProviderSearchListItemFragment_provider">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "TerraformProviderSearchListItemFragment_provider",
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
      "name": "name",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "registryNamespace",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "private",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "TerraformProviderVersion",
      "kind": "LinkedField",
      "name": "latestVersion",
      "plural": false,
      "selections": [
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
        }
      ],
      "storageKey": null
    }
  ],
  "type": "TerraformProvider",
  "abstractKey": null
};

(node as any).hash = "26ea3a4f2864ab6574ec38f91abbff8a";

export default node;
