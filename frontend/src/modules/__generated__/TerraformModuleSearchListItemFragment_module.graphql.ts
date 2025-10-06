/**
 * @generated SignedSource<<561bb93779d8c4228fbd6dddefe44925>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TerraformModuleSearchListItemFragment_module$data = {
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
  readonly system: string;
  readonly " $fragmentType": "TerraformModuleSearchListItemFragment_module";
};
export type TerraformModuleSearchListItemFragment_module$key = {
  readonly " $data"?: TerraformModuleSearchListItemFragment_module$data;
  readonly " $fragmentSpreads": FragmentRefs<"TerraformModuleSearchListItemFragment_module">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "TerraformModuleSearchListItemFragment_module",
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
      "name": "system",
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
      "concreteType": "TerraformModuleVersion",
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
  "type": "TerraformModule",
  "abstractKey": null
};

(node as any).hash = "be93542aec17552e4d5ccbbf52ff659d";

export default node;
