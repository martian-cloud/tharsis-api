/**
 * @generated SignedSource<<c53f6b8b6cecb0adaa9b45b4ab45d92f>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type HomeFragment_activity$data = {
  readonly activityEvents: {
    readonly totalCount: number;
  };
  readonly config: {
    readonly tharsisSupportUrl: string;
  };
  readonly " $fragmentType": "HomeFragment_activity";
};
export type HomeFragment_activity$key = {
  readonly " $data"?: HomeFragment_activity$data;
  readonly " $fragmentSpreads": FragmentRefs<"HomeFragment_activity">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "HomeFragment_activity",
  "selections": [
    {
      "alias": null,
      "args": [
        {
          "kind": "Literal",
          "name": "first",
          "value": 0
        }
      ],
      "concreteType": "ActivityEventConnection",
      "kind": "LinkedField",
      "name": "activityEvents",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "totalCount",
          "storageKey": null
        }
      ],
      "storageKey": "activityEvents(first:0)"
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "Config",
      "kind": "LinkedField",
      "name": "config",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "tharsisSupportUrl",
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Query",
  "abstractKey": null
};

(node as any).hash = "ad9a6f0567e5dd61c8dcc3d45a9c7dde";

export default node;
