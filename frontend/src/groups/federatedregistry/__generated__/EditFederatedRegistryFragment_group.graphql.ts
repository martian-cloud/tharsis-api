/**
 * @generated SignedSource<<72af676d3309cc99593b403c4433e1a4>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type EditFederatedRegistryFragment_group$data = {
  readonly fullPath: string;
  readonly " $fragmentType": "EditFederatedRegistryFragment_group";
};
export type EditFederatedRegistryFragment_group$key = {
  readonly " $data"?: EditFederatedRegistryFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"EditFederatedRegistryFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "EditFederatedRegistryFragment_group",
  "selections": [
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

(node as any).hash = "5a6afa1aff0846b75ceacea1631081d2";

export default node;
