/**
 * @generated SignedSource<<ac85e611ec2cf3ee0f3ef366a0730ddb>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type AdminAreaEmailOutboxListQuery$variables = {
  after?: string | null | undefined;
  ephemeral?: boolean | null | undefined;
  first?: number | null | undefined;
  subjectSearch?: string | null | undefined;
};
export type AdminAreaEmailOutboxListQuery$data = {
  readonly " $fragmentSpreads": FragmentRefs<"AdminAreaEmailOutboxListFragment_outboxes">;
};
export type AdminAreaEmailOutboxListQuery = {
  response: AdminAreaEmailOutboxListQuery$data;
  variables: AdminAreaEmailOutboxListQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "after"
},
v1 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "ephemeral"
},
v2 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "first"
},
v3 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "subjectSearch"
},
v4 = [
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
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/),
      (v2/*: any*/),
      (v3/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "AdminAreaEmailOutboxListQuery",
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
    "argumentDefinitions": [
      (v2/*: any*/),
      (v0/*: any*/),
      (v3/*: any*/),
      (v1/*: any*/)
    ],
    "kind": "Operation",
    "name": "AdminAreaEmailOutboxListQuery",
    "selections": [
      {
        "alias": null,
        "args": (v4/*: any*/),
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
        "args": (v4/*: any*/),
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
    "cacheID": "c9c852fc9d62a8359ef3b3662f2da85f",
    "id": null,
    "metadata": {},
    "name": "AdminAreaEmailOutboxListQuery",
    "operationKind": "query",
    "text": "query AdminAreaEmailOutboxListQuery(\n  $first: Int\n  $after: String\n  $subjectSearch: String\n  $ephemeral: Boolean\n) {\n  ...AdminAreaEmailOutboxListFragment_outboxes\n}\n\nfragment AdminAreaEmailOutboxListFragment_outboxes on Query {\n  emailOutboxItems(first: $first, after: $after, subjectSearch: $subjectSearch, ephemeral: $ephemeral, sort: CREATED_AT_DESC) {\n    totalCount\n    edges {\n      node {\n        id\n        ...AdminAreaEmailOutboxListItemFragment_outbox\n        __typename\n      }\n      cursor\n    }\n    pageInfo {\n      endCursor\n      hasNextPage\n    }\n  }\n}\n\nfragment AdminAreaEmailOutboxListItemFragment_outbox on EmailOutboxItem {\n  id\n  subject\n  emailType\n  ephemeral\n  metadata {\n    createdAt\n  }\n}\n"
  }
};
})();

(node as any).hash = "37b19ae9e91ce9cb8df64d8cdf2a51e6";

export default node;
