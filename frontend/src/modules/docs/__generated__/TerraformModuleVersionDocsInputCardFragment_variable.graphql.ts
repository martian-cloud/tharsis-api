/**
 * @generated SignedSource<<966a1d9111888dc1ccb27b1cbb19336f>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TerraformModuleVersionDocsInputCardFragment_variable$data = {
  readonly default: string | null | undefined;
  readonly description: string;
  readonly name: string;
  readonly required: boolean;
  readonly sensitive: boolean;
  readonly type: string;
  readonly " $fragmentType": "TerraformModuleVersionDocsInputCardFragment_variable";
};
export type TerraformModuleVersionDocsInputCardFragment_variable$key = {
  readonly " $data"?: TerraformModuleVersionDocsInputCardFragment_variable$data;
  readonly " $fragmentSpreads": FragmentRefs<"TerraformModuleVersionDocsInputCardFragment_variable">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "TerraformModuleVersionDocsInputCardFragment_variable",
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
      "name": "default",
      "storageKey": null
    },
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
      "name": "sensitive",
      "storageKey": null
    }
  ],
  "type": "TerraformModuleConfigurationDetailsVariable",
  "abstractKey": null
};

(node as any).hash = "9949e8904aae3ce863eed0d1968fee8f";

export default node;
