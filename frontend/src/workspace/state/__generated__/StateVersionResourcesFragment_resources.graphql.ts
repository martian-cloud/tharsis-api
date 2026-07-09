/**
 * @generated SignedSource<<98c03eaeffb14a6c0708a7194a339690>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type StateVersionResourcesFragment_resources$data = {
  readonly resources: ReadonlyArray<{
    readonly name: string;
    readonly provider: string;
    readonly type: string;
    readonly " $fragmentSpreads": FragmentRefs<"StateVersionResourceListItemFragment_resource">;
  }>;
  readonly " $fragmentType": "StateVersionResourcesFragment_resources";
};
export type StateVersionResourcesFragment_resources$key = {
  readonly " $data"?: StateVersionResourcesFragment_resources$data;
  readonly " $fragmentSpreads": FragmentRefs<"StateVersionResourcesFragment_resources">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "StateVersionResourcesFragment_resources",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "StateVersionResource",
      "kind": "LinkedField",
      "name": "resources",
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
          "name": "provider",
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
          "args": null,
          "kind": "FragmentSpread",
          "name": "StateVersionResourceListItemFragment_resource"
        }
      ],
      "storageKey": null
    }
  ],
  "type": "StateVersionInventory",
  "abstractKey": null
};

(node as any).hash = "d381dce9551354b7a6dee88e262e4cc3";

export default node;
