/**
 * @generated SignedSource<<bc2064495c23c244c92b6d82e32ac75f>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TerraformModuleVersionDocsDataSourcesFragment_dataResources$data = {
  readonly dataResources: ReadonlyArray<{
    readonly name: string;
    readonly type: string;
  }>;
  readonly " $fragmentType": "TerraformModuleVersionDocsDataSourcesFragment_dataResources";
};
export type TerraformModuleVersionDocsDataSourcesFragment_dataResources$key = {
  readonly " $data"?: TerraformModuleVersionDocsDataSourcesFragment_dataResources$data;
  readonly " $fragmentSpreads": FragmentRefs<"TerraformModuleVersionDocsDataSourcesFragment_dataResources">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "TerraformModuleVersionDocsDataSourcesFragment_dataResources",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "TerraformModuleConfigurationDetailsResource",
      "kind": "LinkedField",
      "name": "dataResources",
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

(node as any).hash = "ca11e8671e60b5b8e6b904a3ae5afcef";

export default node;
