/**
 * @generated SignedSource<<6c643411e186fa39af66e269320a2626>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TerraformModuleVersionDocsSidebarFragment_configurationDetails$data = {
  readonly dataResources: ReadonlyArray<{
    readonly name: string;
  }>;
  readonly managedResources: ReadonlyArray<{
    readonly name: string;
  }>;
  readonly outputs: ReadonlyArray<{
    readonly name: string;
  }>;
  readonly readme: string;
  readonly requiredProviders: ReadonlyArray<{
    readonly source: string;
  }>;
  readonly variables: ReadonlyArray<{
    readonly name: string;
  }>;
  readonly " $fragmentType": "TerraformModuleVersionDocsSidebarFragment_configurationDetails";
};
export type TerraformModuleVersionDocsSidebarFragment_configurationDetails$key = {
  readonly " $data"?: TerraformModuleVersionDocsSidebarFragment_configurationDetails$data;
  readonly " $fragmentSpreads": FragmentRefs<"TerraformModuleVersionDocsSidebarFragment_configurationDetails">;
};

const node: ReaderFragment = (function(){
var v0 = [
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "name",
    "storageKey": null
  }
];
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "TerraformModuleVersionDocsSidebarFragment_configurationDetails",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "readme",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "TerraformModuleConfigurationDetailsVariable",
      "kind": "LinkedField",
      "name": "variables",
      "plural": true,
      "selections": (v0/*: any*/),
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "TerraformModuleConfigurationDetailsOutput",
      "kind": "LinkedField",
      "name": "outputs",
      "plural": true,
      "selections": (v0/*: any*/),
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "TerraformModuleConfigurationDetailsResource",
      "kind": "LinkedField",
      "name": "managedResources",
      "plural": true,
      "selections": (v0/*: any*/),
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "TerraformModuleConfigurationDetailsResource",
      "kind": "LinkedField",
      "name": "dataResources",
      "plural": true,
      "selections": (v0/*: any*/),
      "storageKey": null
    },
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
        }
      ],
      "storageKey": null
    }
  ],
  "type": "TerraformModuleConfigurationDetails",
  "abstractKey": null
};
})();

(node as any).hash = "20626aea1db9ed771e1db85b72758516";

export default node;
