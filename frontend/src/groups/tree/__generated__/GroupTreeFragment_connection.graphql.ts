/**
 * @generated SignedSource<<159f7c867c26d9c77ee72a0cedfb573d>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupTreeFragment_connection$data = {
  readonly edges: ReadonlyArray<{
    readonly node: {
      readonly id: string;
      readonly " $fragmentSpreads": FragmentRefs<"GroupTreeListItemFragment_group">;
    } | null | undefined;
  } | null | undefined> | null | undefined;
  readonly " $fragmentType": "GroupTreeFragment_connection";
};
export type GroupTreeFragment_connection$key = {
  readonly " $data"?: GroupTreeFragment_connection$data;
  readonly " $fragmentSpreads": FragmentRefs<"GroupTreeFragment_connection">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "GroupTreeFragment_connection",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "GroupEdge",
      "kind": "LinkedField",
      "name": "edges",
      "plural": true,
      "selections": [
        {
          "alias": null,
          "args": null,
          "concreteType": "Group",
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
              "name": "GroupTreeListItemFragment_group"
            }
          ],
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "GroupConnection",
  "abstractKey": null
};

(node as any).hash = "cda5a11b47b51eb01d52b142c975ee2c";

export default node;
