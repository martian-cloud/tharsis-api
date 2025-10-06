/**
 * @generated SignedSource<<0bdf86c224249aed46fc110be8745aae>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TerraformModuleVersionListFragment_module$data = {
  readonly id: string;
  readonly " $fragmentType": "TerraformModuleVersionListFragment_module";
};
export type TerraformModuleVersionListFragment_module$key = {
  readonly " $data"?: TerraformModuleVersionListFragment_module$data;
  readonly " $fragmentSpreads": FragmentRefs<"TerraformModuleVersionListFragment_module">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "TerraformModuleVersionListFragment_module",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "id",
      "storageKey": null
    }
  ],
  "type": "TerraformModule",
  "abstractKey": null
};

(node as any).hash = "064b28ebc49606f1cb0ff4dd8b56b4ec";

export default node;
