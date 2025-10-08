/**
 * @generated SignedSource<<874be137e00439b3eb79d8cb25af3e13>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TerraformModuleVersionDocsOutputsFragment_outputs$data = {
  readonly outputs: ReadonlyArray<{
    readonly description: string;
    readonly name: string;
    readonly sensitive: boolean;
  }>;
  readonly " $fragmentType": "TerraformModuleVersionDocsOutputsFragment_outputs";
};
export type TerraformModuleVersionDocsOutputsFragment_outputs$key = {
  readonly " $data"?: TerraformModuleVersionDocsOutputsFragment_outputs$data;
  readonly " $fragmentSpreads": FragmentRefs<"TerraformModuleVersionDocsOutputsFragment_outputs">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "TerraformModuleVersionDocsOutputsFragment_outputs",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "TerraformModuleConfigurationDetailsOutput",
      "kind": "LinkedField",
      "name": "outputs",
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
          "name": "description",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "sensitive",
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "TerraformModuleConfigurationDetails",
  "abstractKey": null
};

(node as any).hash = "856e92e8547177ceb773181dd5991e89";

export default node;
