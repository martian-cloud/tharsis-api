/**
 * @generated SignedSource<<ac053d7d16694ec67d6ffb4607fd48de>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type RunListFragment_runConnection$data = {
  readonly edges: ReadonlyArray<{
    readonly node: {
      readonly id: string;
      readonly " $fragmentSpreads": FragmentRefs<"RunListItemFragment_run">;
    } | null | undefined;
  } | null | undefined> | null | undefined;
  readonly totalCount: number;
  readonly " $fragmentType": "RunListFragment_runConnection";
};
export type RunListFragment_runConnection$key = {
  readonly " $data"?: RunListFragment_runConnection$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunListFragment_runConnection">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "RunListFragment_runConnection",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "totalCount",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "RunEdge",
      "kind": "LinkedField",
      "name": "edges",
      "plural": true,
      "selections": [
        {
          "alias": null,
          "args": null,
          "concreteType": "Run",
          "kind": "LinkedField",
          "name": "node",
          "plural": false,
          "selections": [
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "id",
              "storageKey": null
            },
            {
              "args": null,
              "kind": "FragmentSpread",
              "name": "RunListItemFragment_run"
            }
          ],
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "RunConnection",
  "abstractKey": null
};

(node as any).hash = "93b2fc003e001cee10e2b8532f0fbcf7";

export default node;
