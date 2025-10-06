/**
 * @generated SignedSource<<532f4a496354d82670981da876e11dd2>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type StateVersionResourceListItemFragment_resource$data = {
  readonly mode: string;
  readonly module: string;
  readonly name: string;
  readonly provider: string;
  readonly type: string;
  readonly " $fragmentType": "StateVersionResourceListItemFragment_resource";
};
export type StateVersionResourceListItemFragment_resource$key = {
  readonly " $data"?: StateVersionResourceListItemFragment_resource$data;
  readonly " $fragmentSpreads": FragmentRefs<"StateVersionResourceListItemFragment_resource">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "StateVersionResourceListItemFragment_resource",
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
      "name": "provider",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "mode",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "module",
      "storageKey": null
    }
  ],
  "type": "StateVersionResource",
  "abstractKey": null
};

(node as any).hash = "fe94629f49572529ec3c99e78becde0c";

export default node;
