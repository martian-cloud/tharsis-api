/**
 * @generated SignedSource<<72e0d38a8188c2b53cb242518ceede7c>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type AdminAreaEmailOutboxListFragment_outboxes$data = {
  readonly emailOutboxItems: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly id: string;
        readonly " $fragmentSpreads": FragmentRefs<"AdminAreaEmailOutboxListItemFragment_outbox">;
      } | null | undefined;
    } | null | undefined> | null | undefined;
    readonly totalCount: number;
  };
  readonly " $fragmentType": "AdminAreaEmailOutboxListFragment_outboxes";
};
export type AdminAreaEmailOutboxListFragment_outboxes$key = {
  readonly " $data"?: AdminAreaEmailOutboxListFragment_outboxes$data;
  readonly " $fragmentSpreads": FragmentRefs<"AdminAreaEmailOutboxListFragment_outboxes">;
};

import AdminAreaEmailOutboxListPaginationQuery_graphql from './AdminAreaEmailOutboxListPaginationQuery.graphql';

const node: ReaderFragment = (function(){
var v0 = [
  "emailOutboxItems"
];
return {
  "argumentDefinitions": [
    {
      "kind": "RootArgument",
      "name": "after"
    },
    {
      "kind": "RootArgument",
      "name": "ephemeral"
    },
    {
      "kind": "RootArgument",
      "name": "first"
    },
    {
      "kind": "RootArgument",
      "name": "subjectSearch"
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
      "operation": AdminAreaEmailOutboxListPaginationQuery_graphql
    }
  },
  "name": "AdminAreaEmailOutboxListFragment_outboxes",
  "selections": [
    {
      "alias": "emailOutboxItems",
      "args": [
        {
          "kind": "Variable",
          "name": "ephemeral",
          "variableName": "ephemeral"
        },
        {
          "kind": "Literal",
          "name": "sort",
          "value": "CREATED_AT_DESC"
        },
        {
          "kind": "Variable",
          "name": "subjectSearch",
          "variableName": "subjectSearch"
        }
      ],
      "concreteType": "EmailOutboxItemConnection",
      "kind": "LinkedField",
      "name": "__AdminAreaEmailOutboxList_emailOutboxItems_connection",
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
          "concreteType": "EmailOutboxItemEdge",
          "kind": "LinkedField",
          "name": "edges",
          "plural": true,
          "selections": [
            {
              "alias": null,
              "args": null,
              "concreteType": "EmailOutboxItem",
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
                  "name": "AdminAreaEmailOutboxListItemFragment_outbox"
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
      "storageKey": null
    }
  ],
  "type": "Query",
  "abstractKey": null
};
})();

(node as any).hash = "396a7b6db8c2373fc2f23ccb8d97bb46";

export default node;
