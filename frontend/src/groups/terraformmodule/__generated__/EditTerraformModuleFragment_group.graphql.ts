/**
 * @generated SignedSource<<4fcd5ae3b0114bb25b1ec9c56e8a5212>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type EditTerraformModuleFragment_group$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "EditTerraformModuleFragment_group";
};
export type EditTerraformModuleFragment_group$key = {
  readonly " $data"?: EditTerraformModuleFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"EditTerraformModuleFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "EditTerraformModuleFragment_group",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "id",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "fullPath",
      "storageKey": null
    }
  ],
  "type": "Group",
  "abstractKey": null
};

(node as any).hash = "879761152f63e52db6f443188f452b1e";

export default node;
