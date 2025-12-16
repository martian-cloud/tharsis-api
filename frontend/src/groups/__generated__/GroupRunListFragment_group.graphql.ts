/**
 * @generated SignedSource<<9ab40f8052b1221589e738fac0579c51>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupRunListFragment_group$data = {
  readonly group: {
    readonly id: string;
    readonly runs: {
      readonly edges: ReadonlyArray<{
        readonly node: {
          readonly id: string;
        } | null | undefined;
      } | null | undefined> | null | undefined;
      readonly totalCount: number;
      readonly " $fragmentSpreads": FragmentRefs<"RunListFragment_runConnection">;
    };
  } | null | undefined;
  readonly " $fragmentType": "GroupRunListFragment_group";
};
export type GroupRunListFragment_group$key = {
  readonly " $data"?: GroupRunListFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"GroupRunListFragment_group">;
};

import GroupRunListPaginationQuery_graphql from './GroupRunListPaginationQuery.graphql';

const node: ReaderFragment = (function(){
var v0 = [
  "group",
  "runs"
],
v1 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
};
return {
  "argumentDefinitions": [
    {
      "kind": "RootArgument",
      "name": "after"
    },
    {
      "kind": "RootArgument",
      "name": "first"
    },
    {
      "kind": "RootArgument",
      "name": "groupPath"
    },
    {
      "kind": "RootArgument",
      "name": "includeNestedRuns"
    },
    {
      "kind": "RootArgument",
      "name": "workspaceAssessment"
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
      "operation": GroupRunListPaginationQuery_graphql
    }
  },
  "name": "GroupRunListFragment_group",
  "selections": [
    {
      "alias": null,
      "args": [
        {
          "kind": "Variable",
          "name": "fullPath",
          "variableName": "groupPath"
        }
      ],
      "concreteType": "Group",
      "kind": "LinkedField",
      "name": "group",
      "plural": false,
      "selections": [
        (v1/*: any*/),
        {
          "alias": "runs",
          "args": [
            {
              "kind": "Variable",
              "name": "includeNestedRuns",
              "variableName": "includeNestedRuns"
            },
            {
              "kind": "Literal",
              "name": "sort",
              "value": "CREATED_AT_DESC"
            },
            {
              "kind": "Variable",
              "name": "workspaceAssessment",
              "variableName": "workspaceAssessment"
            }
          ],
          "concreteType": "RunConnection",
          "kind": "LinkedField",
          "name": "__GroupRunList_runs_connection",
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
                    (v1/*: any*/),
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
              "name": "RunListFragment_runConnection"
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
      "storageKey": null
    }
  ],
  "type": "Query",
  "abstractKey": null
};
})();

(node as any).hash = "b907b2e0a6b18fb24721d74c9f26681d";

export default node;
