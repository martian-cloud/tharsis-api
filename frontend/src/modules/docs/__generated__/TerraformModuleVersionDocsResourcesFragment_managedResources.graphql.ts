/**
 * @generated SignedSource<<f2263012faa4b47f351df7bd7648fc4a>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TerraformModuleVersionDocsResourcesFragment_managedResources$data = {
  readonly managedResources: ReadonlyArray<{
    readonly name: string;
    readonly type: string;
  }>;
  readonly " $fragmentType": "TerraformModuleVersionDocsResourcesFragment_managedResources";
};
export type TerraformModuleVersionDocsResourcesFragment_managedResources$key = {
  readonly " $data"?: TerraformModuleVersionDocsResourcesFragment_managedResources$data;
  readonly " $fragmentSpreads": FragmentRefs<"TerraformModuleVersionDocsResourcesFragment_managedResources">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "TerraformModuleVersionDocsResourcesFragment_managedResources",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "TerraformModuleConfigurationDetailsResource",
      "kind": "LinkedField",
      "name": "managedResources",
      "plural": true,
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
          "name": "type",
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "TerraformModuleConfigurationDetails",
  "abstractKey": null
};

(node as any).hash = "6db4851d3459455a11070d70d9bdf549";

export default node;
