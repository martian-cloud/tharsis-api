/**
 * @generated SignedSource<<c158d2371b012506a0180747df011592>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TerraformModulesFragment_group$data = {
  readonly " $fragmentSpreads": FragmentRefs<"TerraformModuleListFragment_group">;
  readonly " $fragmentType": "TerraformModulesFragment_group";
};
export type TerraformModulesFragment_group$key = {
  readonly " $data"?: TerraformModulesFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"TerraformModulesFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "TerraformModulesFragment_group",
  "selections": [
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "TerraformModuleListFragment_group"
    }
  ],
  "type": "Group",
  "abstractKey": null
};

(node as any).hash = "37f1c4803d8e2971135086e41162265f";

export default node;
