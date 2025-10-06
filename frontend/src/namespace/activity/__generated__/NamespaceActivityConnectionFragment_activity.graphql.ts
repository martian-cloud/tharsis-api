/**
 * @generated SignedSource<<565349a1d7cfb26560e3210a537abd10>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type NamespaceActivityConnectionFragment_activity$data = {
  readonly activityEvents: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly id: string;
      } | null | undefined;
    } | null | undefined> | null | undefined;
    readonly " $fragmentSpreads": FragmentRefs<"ActivityEventListFragment_connection">;
  };
  readonly " $fragmentType": "NamespaceActivityConnectionFragment_activity";
};
export type NamespaceActivityConnectionFragment_activity$key = {
  readonly " $data"?: NamespaceActivityConnectionFragment_activity$data;
  readonly " $fragmentSpreads": FragmentRefs<"NamespaceActivityConnectionFragment_activity">;
};

import NamespaceActivityPaginationQuery_graphql from './NamespaceActivityPaginationQuery.graphql';

const node: ReaderFragment = (function(){
var v0 = [
  "activityEvents"
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
      "name": "namespacePath"
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
      "operation": NamespaceActivityPaginationQuery_graphql
    }
  },
  "name": "NamespaceActivityConnectionFragment_activity",
  "selections": [
    {
      "alias": "activityEvents",
      "args": [
        {
          "kind": "Literal",
          "name": "includeNested",
          "value": true
        },
        {
          "kind": "Variable",
          "name": "namespacePath",
          "variableName": "namespacePath"
        },
        {
          "kind": "Literal",
          "name": "sort",
          "value": "CREATED_DESC"
        }
      ],
      "concreteType": "ActivityEventConnection",
      "kind": "LinkedField",
      "name": "__NamespaceActivity_activityEvents_connection",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "concreteType": "ActivityEventEdge",
          "kind": "LinkedField",
          "name": "edges",
          "plural": true,
          "selections": [
            {
              "alias": null,
              "args": null,
              "concreteType": "ActivityEvent",
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
          "name": "ActivityEventListFragment_connection"
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

(node as any).hash = "c7845133c7636f00e96f9af6a71f0331";

export default node;
