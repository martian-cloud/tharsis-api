/**
 * @generated SignedSource<<f3d1b631eea0bc415138c6b6b021bcaf>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TerraformProviderVersionListItemFragment_version$data = {
  readonly createdBy: string;
  readonly id: string;
  readonly latest: boolean;
  readonly metadata: {
    readonly createdAt: any;
  };
  readonly provider: {
    readonly name: string;
    readonly registryNamespace: string;
  };
  readonly version: string;
  readonly " $fragmentType": "TerraformProviderVersionListItemFragment_version";
};
export type TerraformProviderVersionListItemFragment_version$key = {
  readonly " $data"?: TerraformProviderVersionListItemFragment_version$data;
  readonly " $fragmentSpreads": FragmentRefs<"TerraformProviderVersionListItemFragment_version">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "TerraformProviderVersionListItemFragment_version",
  "selections": [
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
      "name": "id",
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
      "name": "latest",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "TerraformProvider",
      "kind": "LinkedField",
      "name": "provider",
      "plural": false,
      "selections": [
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
        }
      ],
      "storageKey": null
    }
  ],
  "type": "TerraformProviderVersion",
  "abstractKey": null
};

(node as any).hash = "ae891cf08ddea0206ba49a6dc85c8623";

export default node;
