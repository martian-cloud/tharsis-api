/**
 * @generated SignedSource<<d4659330c528a25d18bb1449c9dba191>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type EmailDeliveryStatus = "ABANDONED" | "ACCEPTED" | "COMPLETED" | "DELAYED" | "FAILED" | "HARD_BOUNCED" | "PENDING" | "SOFT_BOUNCED" | "%future added value";
export type AdminAreaEmailRecipientListPaginationQuery$variables = {
  after?: string | null | undefined;
  deliveryStatuses?: ReadonlyArray<EmailDeliveryStatus> | null | undefined;
  first?: number | null | undefined;
  hasIssues?: boolean | null | undefined;
  hasOpened?: boolean | null | undefined;
  id: string;
  search?: string | null | undefined;
};
export type AdminAreaEmailRecipientListPaginationQuery$data = {
  readonly node: {
    readonly " $fragmentSpreads": FragmentRefs<"AdminAreaEmailRecipientListFragment_outbox">;
  } | null | undefined;
};
export type AdminAreaEmailRecipientListPaginationQuery = {
  response: AdminAreaEmailRecipientListPaginationQuery$data;
  variables: AdminAreaEmailRecipientListPaginationQuery$variables;
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
  "name": "deliveryStatuses"
},
v2 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "first"
},
v3 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "hasIssues"
},
v4 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "hasOpened"
},
v5 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "id"
},
v6 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "search"
},
v7 = [
  {
    "kind": "Variable",
    "name": "id",
    "variableName": "id"
  }
],
v8 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "__typename",
  "storageKey": null
},
v9 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v10 = [
  {
    "kind": "Variable",
    "name": "after",
    "variableName": "after"
  },
  {
    "kind": "Variable",
    "name": "deliveryStatuses",
    "variableName": "deliveryStatuses"
  },
  {
    "kind": "Variable",
    "name": "first",
    "variableName": "first"
  },
  {
    "kind": "Variable",
    "name": "hasIssues",
    "variableName": "hasIssues"
  },
  {
    "kind": "Variable",
    "name": "hasOpened",
    "variableName": "hasOpened"
  },
  {
    "kind": "Variable",
    "name": "search",
    "variableName": "search"
  },
  {
    "kind": "Literal",
    "name": "sort",
    "value": "CREATED_AT_ASC"
  }
];
return {
  "fragment": {
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/),
      (v2/*: any*/),
      (v3/*: any*/),
      (v4/*: any*/),
      (v5/*: any*/),
      (v6/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "AdminAreaEmailRecipientListPaginationQuery",
    "selections": [
      {
        "alias": null,
        "args": (v7/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          {
            "args": null,
            "kind": "FragmentSpread",
            "name": "AdminAreaEmailRecipientListFragment_outbox"
          }
        ],
        "storageKey": null
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/),
      (v2/*: any*/),
      (v3/*: any*/),
      (v4/*: any*/),
      (v6/*: any*/),
      (v5/*: any*/)
    ],
    "kind": "Operation",
    "name": "AdminAreaEmailRecipientListPaginationQuery",
    "selections": [
      {
        "alias": null,
        "args": (v7/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          (v8/*: any*/),
          (v9/*: any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              {
                "alias": null,
                "args": (v10/*: any*/),
                "concreteType": "EmailRecipientConnection",
                "kind": "LinkedField",
                "name": "recipients",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "EmailRecipientEdge",
                    "kind": "LinkedField",
                    "name": "edges",
                    "plural": true,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "EmailRecipient",
                        "kind": "LinkedField",
                        "name": "node",
                        "plural": false,
                        "selections": [
                          (v9/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "address",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "deliveryStatus",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "openedAt",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "clickedAt",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "complainedAt",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "attemptCount",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "availableAt",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "lastAttemptAt",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "failureReason",
                            "storageKey": null
                          },
                          (v8/*: any*/)
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
                "args": (v10/*: any*/),
                "filters": [
                  "search",
                  "deliveryStatuses",
                  "hasOpened",
                  "hasIssues",
                  "sort"
                ],
                "handle": "connection",
                "key": "AdminAreaEmailRecipientList_recipients",
                "kind": "LinkedHandle",
                "name": "recipients"
              }
            ],
            "type": "EmailOutboxItem",
            "abstractKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "d4695f761b7e0b9cfc2bef01c4e2deee",
    "id": null,
    "metadata": {},
    "name": "AdminAreaEmailRecipientListPaginationQuery",
    "operationKind": "query",
    "text": "query AdminAreaEmailRecipientListPaginationQuery(\n  $after: String\n  $deliveryStatuses: [EmailDeliveryStatus!]\n  $first: Int\n  $hasIssues: Boolean\n  $hasOpened: Boolean\n  $search: String\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ...AdminAreaEmailRecipientListFragment_outbox\n    id\n  }\n}\n\nfragment AdminAreaEmailRecipientDialogFragment_recipient on EmailRecipient {\n  id\n  address\n  deliveryStatus\n  attemptCount\n  availableAt\n  lastAttemptAt\n  openedAt\n  clickedAt\n  complainedAt\n  failureReason\n}\n\nfragment AdminAreaEmailRecipientListFragment_outbox on EmailOutboxItem {\n  recipients(first: $first, after: $after, search: $search, deliveryStatuses: $deliveryStatuses, hasOpened: $hasOpened, hasIssues: $hasIssues, sort: CREATED_AT_ASC) {\n    edges {\n      node {\n        id\n        ...AdminAreaEmailRecipientListItemFragment_recipient\n        __typename\n      }\n      cursor\n    }\n    pageInfo {\n      endCursor\n      hasNextPage\n    }\n  }\n  id\n}\n\nfragment AdminAreaEmailRecipientListItemFragment_recipient on EmailRecipient {\n  address\n  deliveryStatus\n  openedAt\n  clickedAt\n  complainedAt\n  ...AdminAreaEmailRecipientDialogFragment_recipient\n}\n"
  }
};
})();

(node as any).hash = "c1dcd3238fb8c759568989536b4cab44";

export default node;
