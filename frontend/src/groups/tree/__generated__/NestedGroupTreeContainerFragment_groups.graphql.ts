/**
 * @generated SignedSource<<4707a7c0a9e399cb81ff8a0886af3a9f>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type NestedGroupTreeContainerFragment_groups$data = {
  readonly groups: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly id: string;
      } | null | undefined;
    } | null | undefined> | null | undefined;
    readonly " $fragmentSpreads": FragmentRefs<"GroupTreeFragment_connection">;
  };
  readonly " $fragmentType": "NestedGroupTreeContainerFragment_groups";
};
export type NestedGroupTreeContainerFragment_groups$key = {
  readonly " $data"?: NestedGroupTreeContainerFragment_groups$data;
  readonly " $fragmentSpreads": FragmentRefs<"NestedGroupTreeContainerFragment_groups">;
};

import NestedGroupsListPaginationQuery_graphql from './NestedGroupsListPaginationQuery.graphql';

const node: ReaderFragment = (function(){
var v0 = [
  "groups"
];
return {
  "argumentDefinitions": [
    {
      "kind": "RootArgument",
      "name": "after"
    },
    {
      "kind": "RootArgument",
      "name": "before"
    },
    {
      "kind": "RootArgument",
      "name": "first"
    },
    {
      "kind": "RootArgument",
      "name": "last"
    },
    {
      "kind": "RootArgument",
      "name": "parentPath"
    }
  ],
  "kind": "Fragment",
  "metadata": {
    "connection": [
      {
        "count": null,
        "cursor": null,
        "direction": "bidirectional",
        "path": (v0/*: any*/)
      }
    ],
    "refetch": {
      "connection": {
        "forward": {
          "count": "first",
          "cursor": "after"
        },
        "backward": {
          "count": "last",
          "cursor": "before"
        },
        "path": (v0/*: any*/)
      },
      "fragmentPathInResult": [],
      "operation": NestedGroupsListPaginationQuery_graphql
    }
  },
  "name": "NestedGroupTreeContainerFragment_groups",
  "selections": [
    {
      "alias": "groups",
      "args": [
        {
          "kind": "Variable",
          "name": "parentPath",
          "variableName": "parentPath"
        },
        {
          "kind": "Literal",
          "name": "sort",
          "value": "FULL_PATH_ASC"
        }
      ],
      "concreteType": "GroupConnection",
      "kind": "LinkedField",
      "name": "__NestedGroupTreeContainer_groups_connection",
      "plural": false,
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
          "name": "GroupTreeFragment_connection"
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
            },
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "hasPreviousPage",
              "storageKey": null
            },
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "startCursor",
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

(node as any).hash = "bfe3ab06e5e47a08f86f982f8cbd8337";

export default node;
