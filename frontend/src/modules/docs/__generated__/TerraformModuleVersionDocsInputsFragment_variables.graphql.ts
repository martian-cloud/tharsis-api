/**
 * @generated SignedSource<<9d0a3c5aaf1e8bfa8ca5977fe2c6b2ca>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TerraformModuleVersionDocsInputsFragment_variables$data = {
  readonly variables: ReadonlyArray<{
    readonly name: string;
    readonly required: boolean;
    readonly " $fragmentSpreads": FragmentRefs<"TerraformModuleVersionDocsInputCardFragment_variable">;
  }>;
  readonly " $fragmentType": "TerraformModuleVersionDocsInputsFragment_variables";
};
export type TerraformModuleVersionDocsInputsFragment_variables$key = {
  readonly " $data"?: TerraformModuleVersionDocsInputsFragment_variables$data;
  readonly " $fragmentSpreads": FragmentRefs<"TerraformModuleVersionDocsInputsFragment_variables">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "TerraformModuleVersionDocsInputsFragment_variables",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "TerraformModuleConfigurationDetailsVariable",
      "kind": "LinkedField",
      "name": "variables",
      "plural": true,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "required",
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
          "args": null,
          "kind": "FragmentSpread",
          "name": "TerraformModuleVersionDocsInputCardFragment_variable"
        }
      ],
      "storageKey": null
    }
  ],
  "type": "TerraformModuleConfigurationDetails",
  "abstractKey": null
};

(node as any).hash = "8fdd5352257cc9d6f73f224f80a56053";

export default node;
