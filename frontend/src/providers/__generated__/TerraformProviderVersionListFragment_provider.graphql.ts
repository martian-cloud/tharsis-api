/**
 * @generated SignedSource<<a6fcbf3fd3f6b41ff5a5d8a416dd6351>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TerraformProviderVersionListFragment_provider$data = {
  readonly id: string;
  readonly " $fragmentType": "TerraformProviderVersionListFragment_provider";
};
export type TerraformProviderVersionListFragment_provider$key = {
  readonly " $data"?: TerraformProviderVersionListFragment_provider$data;
  readonly " $fragmentSpreads": FragmentRefs<"TerraformProviderVersionListFragment_provider">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "TerraformProviderVersionListFragment_provider",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "id",
      "storageKey": null
    }
  ],
  "type": "TerraformProvider",
  "abstractKey": null
};

(node as any).hash = "5312eb7db85031f5d933e961e508a1ad";

export default node;
