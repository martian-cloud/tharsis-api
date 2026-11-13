/**
 * @generated SignedSource<<b3bc906f21593e2ed85841024ed02e2f>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type AdminAreaAnnouncementListQuery$variables = {
  after?: string | null | undefined;
  first: number;
};
export type AdminAreaAnnouncementListQuery$data = {
  readonly " $fragmentSpreads": FragmentRefs<"AdminAreaAnnouncementListFragment_announcements">;
};
export type AdminAreaAnnouncementListQuery = {
  response: AdminAreaAnnouncementListQuery$data;
  variables: AdminAreaAnnouncementListQuery$variables;
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
  "name": "first"
},
v2 = [
  {
    "kind": "Variable",
    "name": "after",
    "variableName": "after"
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
  }
];
return {
  "fragment": {
    "argumentDefinitions": [
      (v0/*:: as any*/),
      (v1/*:: as any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "AdminAreaAnnouncementListQuery",
    "selections": [
      {
        "args": null,
        "kind": "FragmentSpread",
        "name": "AdminAreaAnnouncementListFragment_announcements"
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [
      (v1/*:: as any*/),
      (v0/*:: as any*/)
    ],
    "kind": "Operation",
    "name": "AdminAreaAnnouncementListQuery",
    "selections": [
      {
        "alias": null,
        "args": (v2/*:: as any*/),
        "concreteType": "AnnouncementConnection",
        "kind": "LinkedField",
        "name": "announcements",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "AnnouncementEdge",
            "kind": "LinkedField",
            "name": "edges",
            "plural": true,
            "selections": [
              {
                "alias": null,
                "args": null,
                "concreteType": "Announcement",
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
                    "name": "message",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "type",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "dismissible",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "startTime",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "endTime",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "active",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "expired",
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
        "args": (v2/*:: as any*/),
        "filters": [
          "sort"
        ],
        "handle": "connection",
        "key": "AdminAreaAnnouncementList_announcements",
        "kind": "LinkedHandle",
        "name": "announcements"
      }
    ]
  },
  "params": {
    "cacheID": "be23660cce8ce28d6a4e61c7e826ff17",
    "id": null,
    "metadata": {},
    "name": "AdminAreaAnnouncementListQuery",
    "operationKind": "query",
    "text": "query AdminAreaAnnouncementListQuery(\n  $first: Int!\n  $after: String\n) {\n  ...AdminAreaAnnouncementListFragment_announcements\n}\n\nfragment AdminAreaAnnouncementListFragment_announcements on Query {\n  announcements(first: $first, after: $after, sort: CREATED_AT_DESC) {\n    edges {\n      node {\n        id\n        message\n        type\n        dismissible\n        startTime\n        endTime\n        active\n        expired\n        __typename\n      }\n      cursor\n    }\n    pageInfo {\n      endCursor\n      hasNextPage\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "48d25c1ae71da5f3f38bb1105e1b2e3a";

export default node;
