/**
 * @generated SignedSource<<0d936fd51e4ea91f20887e2b335e52f6>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type StateVersionResourcesFragment_resources$data = {
  readonly resources: ReadonlyArray<{
    readonly mode: string;
    readonly module: string;
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
          "name": "mode",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "module",
          "storageKey": null
        },
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

(node as any).hash = "d33e991342df7e4bb81551d10cd8d153";

export default node;
