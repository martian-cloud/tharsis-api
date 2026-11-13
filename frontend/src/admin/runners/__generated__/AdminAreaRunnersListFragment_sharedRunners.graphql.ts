/**
 * @generated SignedSource<<4cbb0a8ff5fe132d92b1cc5d2761bb07>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type AdminAreaRunnersListFragment_sharedRunners$data = {
  readonly sharedRunners: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly id: string;
      } | null | undefined;
    } | null | undefined> | null | undefined;
    readonly " $fragmentSpreads": FragmentRefs<"RunnerListFragment_runners">;
  };
  readonly " $fragmentType": "AdminAreaRunnersListFragment_sharedRunners";
};
export type AdminAreaRunnersListFragment_sharedRunners$key = {
  readonly " $data"?: AdminAreaRunnersListFragment_sharedRunners$data;
  readonly " $fragmentSpreads": FragmentRefs<"AdminAreaRunnersListFragment_sharedRunners">;
};

import AdminAreaRunnersListPaginationQuery_graphql from './AdminAreaRunnersListPaginationQuery.graphql';

const node: ReaderFragment = (function(){
var v0 = [
  "sharedRunners"
];
return {
  "argumentDefinitions": [
    {
      "kind": "RootArgument",
      "name": "after"
    },
    {
      "kind": "RootArgument",
      "name": "first"
    }
  ],
  "kind": "Fragment",
  "metadata": {
    "connection": [
      {
        "count": "first",
        "cursor": "after",
        "direction": "forward",
        "path": (v0/*:: as any*/)
      }
    ],
    "refetch": {
      "connection": {
        "forward": {
          "count": "first",
          "cursor": "after"
        },
        "backward": null,
        "path": (v0/*:: as any*/)
      },
      "fragmentPathInResult": [],
      "operation": AdminAreaRunnersListPaginationQuery_graphql
    }
  },
  "name": "AdminAreaRunnersListFragment_sharedRunners",
  "selections": [
    {
      "alias": "sharedRunners",
      "args": null,
      "concreteType": "RunnerConnection",
      "kind": "LinkedField",
      "name": "__AdminAreaRunnersList_sharedRunners_connection",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "concreteType": "RunnerEdge",
          "kind": "LinkedField",
          "name": "edges",
          "plural": true,
          "selections": [
            {
              "alias": null,
              "args": null,
              "concreteType": "Runner",
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
                  "alias": null,
                  "args": null,
                  "kind": "ScalarField",
                  "name": "__typename",
                  "storageKey": null
                }
              ],
              "storageKey": null
            },
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "cursor",
              "storageKey": null
            }
          ],
          "storageKey": null
        },
        {
          "args": null,
          "kind": "FragmentSpread",
          "name": "RunnerListFragment_runners"
        },
        {
          "alias": null,
          "args": null,
          "concreteType": "PageInfo",
          "kind": "LinkedField",
          "name": "pageInfo",
          "plural": false,
          "selections": [
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "endCursor",
              "storageKey": null
            },
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "hasNextPage",
              "storageKey": null
            }
          ],
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Query",
  "abstractKey": null
};
})();

(node as any).hash = "4b6650cee7cea445fdea51ab16c62fb1";

export default node;
