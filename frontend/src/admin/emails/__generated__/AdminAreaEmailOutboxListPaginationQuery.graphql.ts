/**
 * @generated SignedSource<<0117d87bcd49634cccd07fe9e6a92a45>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type AdminAreaEmailOutboxListPaginationQuery$variables = {
  after?: string | null | undefined;
  ephemeral?: boolean | null | undefined;
  first?: number | null | undefined;
  subjectSearch?: string | null | undefined;
};
export type AdminAreaEmailOutboxListPaginationQuery$data = {
  readonly " $fragmentSpreads": FragmentRefs<"AdminAreaEmailOutboxListFragment_outboxes">;
};
export type AdminAreaEmailOutboxListPaginationQuery = {
  response: AdminAreaEmailOutboxListPaginationQuery$data;
  variables: AdminAreaEmailOutboxListPaginationQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "after"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "ephemeral"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "first"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "subjectSearch"
  }
],
v1 = [
  {
    "kind": "Variable",
    "name": "after",
    "variableName": "after"
  },
  {
    "kind": "Variable",
    "name": "ephemeral",
    "variableName": "ephemeral"
  },
  {
    "kind": "Variable",
    "name": "first",
    "variableName": "first"
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
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "AdminAreaEmailOutboxListPaginationQuery",
    "selections": [
      {
        "args": null,
        "kind": "FragmentSpread",
        "name": "AdminAreaEmailOutboxListFragment_outboxes"
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "AdminAreaEmailOutboxListPaginationQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "EmailOutboxItemConnection",
        "kind": "LinkedField",
        "name": "emailOutboxItems",
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
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "subject",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "emailType",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "ephemeral",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "ResourceMetadata",
                    "kind": "LinkedField",
                    "name": "metadata",
                    "plural": false,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "createdAt",
                        "storageKey": null
                      }
                    ],
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
      },
      {
        "alias": null,
        "args": (v1/*: any*/),
        "filters": [
          "subjectSearch",
          "ephemeral",
          "sort"
        ],
        "handle": "connection",
        "key": "AdminAreaEmailOutboxList_emailOutboxItems",
        "kind": "LinkedHandle",
        "name": "emailOutboxItems"
      }
    ]
  },
  "params": {
    "cacheID": "14c716490e96257dfa2b3d857a612a78",
    "id": null,
    "metadata": {},
    "name": "AdminAreaEmailOutboxListPaginationQuery",
    "operationKind": "query",
    "text": "query AdminAreaEmailOutboxListPaginationQuery(\n  $after: String\n  $ephemeral: Boolean\n  $first: Int\n  $subjectSearch: String\n) {\n  ...AdminAreaEmailOutboxListFragment_outboxes\n}\n\nfragment AdminAreaEmailOutboxListFragment_outboxes on Query {\n  emailOutboxItems(first: $first, after: $after, subjectSearch: $subjectSearch, ephemeral: $ephemeral, sort: CREATED_AT_DESC) {\n    totalCount\n    edges {\n      node {\n        id\n        ...AdminAreaEmailOutboxListItemFragment_outbox\n        __typename\n      }\n      cursor\n    }\n    pageInfo {\n      endCursor\n      hasNextPage\n    }\n  }\n}\n\nfragment AdminAreaEmailOutboxListItemFragment_outbox on EmailOutboxItem {\n  id\n  subject\n  emailType\n  ephemeral\n  metadata {\n    createdAt\n  }\n}\n"
  }
};
})();

(node as any).hash = "396a7b6db8c2373fc2f23ccb8d97bb46";

export default node;
