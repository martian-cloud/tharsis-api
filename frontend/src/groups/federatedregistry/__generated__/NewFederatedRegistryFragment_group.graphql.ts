/**
 * @generated SignedSource<<d4fe50d3894423cc53a7982e03d28097>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type NewFederatedRegistryFragment_group$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "NewFederatedRegistryFragment_group";
};
export type NewFederatedRegistryFragment_group$key = {
  readonly " $data"?: NewFederatedRegistryFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"NewFederatedRegistryFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "NewFederatedRegistryFragment_group",
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

(node as any).hash = "231603d1c520f5c431cba34b8feb9bde";

export default node;
