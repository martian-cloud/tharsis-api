/**
 * @generated SignedSource<<4f3a894c68809945d540e310cf0df518>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type HomeRunListFragment_runs$data = {
  readonly runs: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly id: string;
        readonly " $fragmentSpreads": FragmentRefs<"HomeRunListItemFragment_run">;
      } | null | undefined;
    } | null | undefined> | null | undefined;
    readonly totalCount: number;
  };
  readonly " $fragmentType": "HomeRunListFragment_runs";
};
export type HomeRunListFragment_runs$key = {
  readonly " $data"?: HomeRunListFragment_runs$data;
  readonly " $fragmentSpreads": FragmentRefs<"HomeRunListFragment_runs">;
};

import HomeRunListPaginationQuery_graphql from './HomeRunListPaginationQuery.graphql';

const node: ReaderFragment = (function(){
var v0 = [
  "runs"
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
        "path": (v0/*: any*/)
      }
    ],
    "refetch": {
      "connection": {
        "forward": {
          "count": "first",
          "cursor": "after"
        },
        "backward": null,
        "path": (v0/*: any*/)
      },
      "fragmentPathInResult": [],
      "operation": HomeRunListPaginationQuery_graphql
    }
  },
  "name": "HomeRunListFragment_runs",
  "selections": [
    {
      "alias": "runs",
      "args": [
        {
          "kind": "Literal",
          "name": "sort",
          "value": "UPDATED_AT_DESC"
        },
        {
          "kind": "Literal",
          "name": "workspaceAssessment",
          "value": false
        }
      ],
      "concreteType": "RunConnection",
      "kind": "LinkedField",
      "name": "__HomeRunList_runs_connection",
      "plural": false,
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
                  "name": "HomeRunListItemFragment_run"
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
      "storageKey": "__HomeRunList_runs_connection(sort:\"UPDATED_AT_DESC\",workspaceAssessment:false)"
    }
  ],
  "type": "Query",
  "abstractKey": null
};
})();

(node as any).hash = "c16569127c66edb21ba55ee4130d0d96";

export default node;
