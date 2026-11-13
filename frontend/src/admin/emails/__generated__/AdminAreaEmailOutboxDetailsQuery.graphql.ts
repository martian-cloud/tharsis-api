/**
 * @generated SignedSource<<a91c1dd7d6f73caad7958f5bc663ced9>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type EmailDeliveryStatus = "ABANDONED" | "ACCEPTED" | "COMPLETED" | "DELAYED" | "FAILED" | "HARD_BOUNCED" | "PENDING" | "SOFT_BOUNCED" | "%future added value";
export type EmailOutboxItemStatus = "COMPLETED" | "FAILED" | "PREPARING" | "READY" | "%future added value";
export type AdminAreaEmailOutboxDetailsQuery$variables = {
  after?: string | null | undefined;
  deliveryStatuses?: ReadonlyArray<EmailDeliveryStatus> | null | undefined;
  first?: number | null | undefined;
  hasIssues?: boolean | null | undefined;
  hasOpened?: boolean | null | undefined;
  id: string;
  search?: string | null | undefined;
};
export type AdminAreaEmailOutboxDetailsQuery$data = {
  readonly config: {
    readonly emailEphemeralRetentionDays: number;
  };
  readonly node: {
    readonly emailOutboxItem: {
      readonly emailType: string;
      readonly ephemeral: boolean;
      readonly id: string;
      readonly metadata: {
        readonly createdAt: any;
        readonly trn: string;
      };
      readonly recipientStats: {
        readonly clicked: number;
        readonly delivered: number;
        readonly issues: number;
        readonly opened: number;
        readonly total: number;
      };
      readonly status: EmailOutboxItemStatus;
      readonly subject: string;
      readonly " $fragmentSpreads": FragmentRefs<"AdminAreaEmailRecipientListFragment_outbox">;
    } | null | undefined;
  } | null | undefined;
};
export type AdminAreaEmailOutboxDetailsQuery = {
  response: AdminAreaEmailOutboxDetailsQuery$data;
  variables: AdminAreaEmailOutboxDetailsQuery$variables;
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
v7 = {
  "alias": null,
  "args": null,
  "concreteType": "Config",
  "kind": "LinkedField",
  "name": "config",
  "plural": false,
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "emailEphemeralRetentionDays",
      "storageKey": null
    }
  ],
  "storageKey": null
},
v8 = [
  {
    "kind": "Variable",
    "name": "id",
    "variableName": "id"
  }
],
v9 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v10 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "subject",
  "storageKey": null
},
v11 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "emailType",
  "storageKey": null
},
v12 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "ephemeral",
  "storageKey": null
},
v13 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "status",
  "storageKey": null
},
v14 = {
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
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "trn",
      "storageKey": null
    }
  ],
  "storageKey": null
},
v15 = {
  "alias": null,
  "args": null,
  "concreteType": "EmailRecipientStats",
  "kind": "LinkedField",
  "name": "recipientStats",
  "plural": false,
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "total",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "delivered",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "opened",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "clicked",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "issues",
      "storageKey": null
    }
  ],
  "storageKey": null
},
v16 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "__typename",
  "storageKey": null
},
v17 = [
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
      (v0/*:: as any*/),
      (v1/*:: as any*/),
      (v2/*:: as any*/),
      (v3/*:: as any*/),
      (v4/*:: as any*/),
      (v5/*:: as any*/),
      (v6/*:: as any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "AdminAreaEmailOutboxDetailsQuery",
    "selections": [
      (v7/*:: as any*/),
      {
        "alias": null,
        "args": (v8/*:: as any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          {
            "fragment": {
              "kind": "InlineFragment",
              "selections": [
                (v9/*:: as any*/),
                (v10/*:: as any*/),
                (v11/*:: as any*/),
                (v12/*:: as any*/),
                (v13/*:: as any*/),
                (v14/*:: as any*/),
                (v15/*:: as any*/),
                {
                  "args": null,
                  "kind": "FragmentSpread",
                  "name": "AdminAreaEmailRecipientListFragment_outbox"
                }
              ],
              "type": "EmailOutboxItem",
              "abstractKey": null
            },
            "kind": "AliasedInlineFragmentSpread",
            "name": "emailOutboxItem"
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
      (v5/*:: as any*/),
      (v2/*:: as any*/),
      (v0/*:: as any*/),
      (v6/*:: as any*/),
      (v1/*:: as any*/),
      (v4/*:: as any*/),
      (v3/*:: as any*/)
    ],
    "kind": "Operation",
    "name": "AdminAreaEmailOutboxDetailsQuery",
    "selections": [
      (v7/*:: as any*/),
      {
        "alias": null,
        "args": (v8/*:: as any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          (v16/*:: as any*/),
          (v9/*:: as any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              (v10/*:: as any*/),
              (v11/*:: as any*/),
              (v12/*:: as any*/),
              (v13/*:: as any*/),
              (v14/*:: as any*/),
              (v15/*:: as any*/),
              {
                "alias": null,
                "args": (v17/*:: as any*/),
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
                          (v9/*:: as any*/),
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
                          (v16/*:: as any*/)
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
                "args": (v17/*:: as any*/),
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
    "cacheID": "4c4b8cf71ae942120ed013d79d205854",
    "id": null,
    "metadata": {},
    "name": "AdminAreaEmailOutboxDetailsQuery",
    "operationKind": "query",
    "text": "query AdminAreaEmailOutboxDetailsQuery(\n  $id: String!\n  $first: Int\n  $after: String\n  $search: String\n  $deliveryStatuses: [EmailDeliveryStatus!]\n  $hasOpened: Boolean\n  $hasIssues: Boolean\n) {\n  config {\n    emailEphemeralRetentionDays\n  }\n  node(id: $id) {\n    __typename\n    ... on EmailOutboxItem {\n      id\n      subject\n      emailType\n      ephemeral\n      status\n      metadata {\n        createdAt\n        trn\n      }\n      recipientStats {\n        total\n        delivered\n        opened\n        clicked\n        issues\n      }\n      ...AdminAreaEmailRecipientListFragment_outbox\n    }\n    id\n  }\n}\n\nfragment AdminAreaEmailRecipientDialogFragment_recipient on EmailRecipient {\n  id\n  address\n  deliveryStatus\n  attemptCount\n  availableAt\n  lastAttemptAt\n  openedAt\n  clickedAt\n  complainedAt\n  failureReason\n}\n\nfragment AdminAreaEmailRecipientListFragment_outbox on EmailOutboxItem {\n  recipients(first: $first, after: $after, search: $search, deliveryStatuses: $deliveryStatuses, hasOpened: $hasOpened, hasIssues: $hasIssues, sort: CREATED_AT_ASC) {\n    edges {\n      node {\n        id\n        ...AdminAreaEmailRecipientListItemFragment_recipient\n        __typename\n      }\n      cursor\n    }\n    pageInfo {\n      endCursor\n      hasNextPage\n    }\n  }\n  id\n}\n\nfragment AdminAreaEmailRecipientListItemFragment_recipient on EmailRecipient {\n  address\n  deliveryStatus\n  openedAt\n  clickedAt\n  complainedAt\n  ...AdminAreaEmailRecipientDialogFragment_recipient\n}\n"
  }
};
})();

(node as any).hash = "70b170788c0c35863fccf9a21ff27706";

export default node;
