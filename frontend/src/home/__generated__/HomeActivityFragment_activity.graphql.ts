/**
 * @generated SignedSource<<3218084c034c77dbb7e4387932deeb74>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment, RefetchableFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type HomeActivityFragment_activity$data = {
  readonly activityEvents: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly id: string;
      } | null;
    } | null> | null;
    readonly " $fragmentSpreads": FragmentRefs<"ActivityEventListFragment_connection">;
  };
  readonly " $fragmentType": "HomeActivityFragment_activity";
};
export type HomeActivityFragment_activity$key = {
  readonly " $data"?: HomeActivityFragment_activity$data;
  readonly " $fragmentSpreads": FragmentRefs<"HomeActivityFragment_activity">;
};

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
      "operation": require('./HomeActivityPaginationQuery.graphql')
    }
  },
  "name": "HomeActivityFragment_activity",
  "selections": [
    {
      "alias": "activityEvents",
      "args": [
        {
          "kind": "Literal",
          "name": "sort",
          "value": "CREATED_DESC"
        }
      ],
      "concreteType": "ActivityEventConnection",
      "kind": "LinkedField",
      "name": "__HomeActivity_activityEvents_connection",
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
      "storageKey": "__HomeActivity_activityEvents_connection(sort:\"CREATED_DESC\")"
    }
  ],
  "type": "Query",
  "abstractKey": null
};
})();

(node as any).hash = "21cd59c856327176218d5eb6f7cef56d";

export default node;
