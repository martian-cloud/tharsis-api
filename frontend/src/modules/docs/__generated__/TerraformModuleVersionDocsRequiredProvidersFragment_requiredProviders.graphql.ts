/**
 * @generated SignedSource<<e83b2330f942dd60b1e1ce03cc1343a5>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TerraformModuleVersionDocsRequiredProvidersFragment_requiredProviders$data = {
  readonly requiredProviders: ReadonlyArray<{
    readonly source: string;
    readonly versionConstraints: ReadonlyArray<string>;
  }>;
  readonly " $fragmentType": "TerraformModuleVersionDocsRequiredProvidersFragment_requiredProviders";
};
export type TerraformModuleVersionDocsRequiredProvidersFragment_requiredProviders$key = {
  readonly " $data"?: TerraformModuleVersionDocsRequiredProvidersFragment_requiredProviders$data;
  readonly " $fragmentSpreads": FragmentRefs<"TerraformModuleVersionDocsRequiredProvidersFragment_requiredProviders">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "TerraformModuleVersionDocsRequiredProvidersFragment_requiredProviders",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "TerraformModuleConfigurationDetailsProviderRequirement",
      "kind": "LinkedField",
      "name": "requiredProviders",
      "plural": true,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "source",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "versionConstraints",
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "TerraformModuleConfigurationDetails",
  "abstractKey": null
};

(node as any).hash = "fb001c64fbf25df9bc19f9c2d73e704f";

export default node;
