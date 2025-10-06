/**
 * @generated SignedSource<<f4b9f47830aae750af01a085a9b1ae50>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TerraformModuleVersionDocsFragment_configurationDetails$data = {
  readonly readme: string;
  readonly " $fragmentSpreads": FragmentRefs<"TerraformModuleVersionDocsDataSourcesFragment_dataResources" | "TerraformModuleVersionDocsInputsFragment_variables" | "TerraformModuleVersionDocsOutputsFragment_outputs" | "TerraformModuleVersionDocsRequiredProvidersFragment_requiredProviders" | "TerraformModuleVersionDocsResourcesFragment_managedResources" | "TerraformModuleVersionDocsSidebarFragment_configurationDetails">;
  readonly " $fragmentType": "TerraformModuleVersionDocsFragment_configurationDetails";
};
export type TerraformModuleVersionDocsFragment_configurationDetails$key = {
  readonly " $data"?: TerraformModuleVersionDocsFragment_configurationDetails$data;
  readonly " $fragmentSpreads": FragmentRefs<"TerraformModuleVersionDocsFragment_configurationDetails">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "TerraformModuleVersionDocsFragment_configurationDetails",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "readme",
      "storageKey": null
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "TerraformModuleVersionDocsSidebarFragment_configurationDetails"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "TerraformModuleVersionDocsInputsFragment_variables"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "TerraformModuleVersionDocsOutputsFragment_outputs"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "TerraformModuleVersionDocsResourcesFragment_managedResources"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "TerraformModuleVersionDocsDataSourcesFragment_dataResources"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "TerraformModuleVersionDocsRequiredProvidersFragment_requiredProviders"
    }
  ],
  "type": "TerraformModuleConfigurationDetails",
  "abstractKey": null
};

(node as any).hash = "c2167dca4d0ac3bbb54e3c8ce9a0c7d1";

export default node;
