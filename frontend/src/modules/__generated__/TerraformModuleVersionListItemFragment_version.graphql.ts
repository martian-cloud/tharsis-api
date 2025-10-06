/**
 * @generated SignedSource<<16e3c9fae286709fd3b9b54d21aa101f>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TerraformModuleVersionListItemFragment_version$data = {
  readonly createdBy: string;
  readonly id: string;
  readonly latest: boolean;
  readonly metadata: {
    readonly createdAt: any;
  };
  readonly module: {
    readonly name: string;
    readonly registryNamespace: string;
    readonly system: string;
  };
  readonly version: string;
  readonly " $fragmentType": "TerraformModuleVersionListItemFragment_version";
};
export type TerraformModuleVersionListItemFragment_version$key = {
  readonly " $data"?: TerraformModuleVersionListItemFragment_version$data;
  readonly " $fragmentSpreads": FragmentRefs<"TerraformModuleVersionListItemFragment_version">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "TerraformModuleVersionListItemFragment_version",
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
      "concreteType": "TerraformModule",
      "kind": "LinkedField",
      "name": "module",
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
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "system",
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "TerraformModuleVersion",
  "abstractKey": null
};

(node as any).hash = "9470134307fe5aea865b6711b94aacb2";

export default node;
