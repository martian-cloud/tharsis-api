/**
 * @generated SignedSource<<18c1b2ae368f59a215f617e2f47a5156>>
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
  "type": "StateVersion",
  "abstractKey": null
};

(node as any).hash = "b9fdeb5a74818238b5a76d4cd284cd92";

export default node;
