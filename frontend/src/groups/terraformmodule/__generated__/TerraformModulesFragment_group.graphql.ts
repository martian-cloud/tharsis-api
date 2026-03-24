/**
 * @generated SignedSource<<b6e3f2f42ddcc8204d26baff02785e86>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TerraformModulesFragment_group$data = {
  readonly " $fragmentSpreads": FragmentRefs<"EditTerraformModuleFragment_group" | "TerraformModuleListFragment_group">;
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
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "EditTerraformModuleFragment_group"
    }
  ],
  "type": "Group",
  "abstractKey": null
};

(node as any).hash = "8c3ec4d43b348fb114b2c102bad9f377";

export default node;
