/**
 * @generated SignedSource<<96af25768b4ff8e77d4c158e672a1b14>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type AnnouncementType = "ERROR" | "INFO" | "SUCCESS" | "WARNING" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type AdminAreaAnnouncementListFragment_announcements$data = {
  readonly announcements: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly active: boolean;
        readonly dismissible: boolean;
        readonly endTime: any | null | undefined;
        readonly expired: boolean;
        readonly id: string;
        readonly message: string;
        readonly startTime: any;
        readonly type: AnnouncementType;
      } | null | undefined;
    } | null | undefined> | null | undefined;
  };
  readonly " $fragmentType": "AdminAreaAnnouncementListFragment_announcements";
};
export type AdminAreaAnnouncementListFragment_announcements$key = {
  readonly " $data"?: AdminAreaAnnouncementListFragment_announcements$data;
  readonly " $fragmentSpreads": FragmentRefs<"AdminAreaAnnouncementListFragment_announcements">;
};

import AnnouncementPaginationQuery_graphql from './AnnouncementPaginationQuery.graphql';

const node: ReaderFragment = (function(){
var v0 = [
  "announcements"
];
return {
  "argumentDefinitions": [
    {
      "kind": "RootArgument",
      "name": "after"
    },
    {
      "kind": "RootArgument",
      "name": "first"
    }
  ],
  "kind": "Fragment",
  "metadata": {
    "connection": [
      {
        "count": "first",
        "cursor": "after",
        "direction": "forward",
        "path": (v0/*:: as any*/)
      }
    ],
    "refetch": {
      "connection": {
        "forward": {
          "count": "first",
          "cursor": "after"
        },
        "backward": null,
        "path": (v0/*:: as any*/)
      },
      "fragmentPathInResult": [],
      "operation": AnnouncementPaginationQuery_graphql
    }
  },
  "name": "AdminAreaAnnouncementListFragment_announcements",
  "selections": [
    {
      "alias": "announcements",
      "args": [
        {
          "kind": "Literal",
          "name": "sort",
          "value": "CREATED_AT_DESC"
        }
      ],
      "concreteType": "AnnouncementConnection",
      "kind": "LinkedField",
      "name": "__AdminAreaAnnouncementList_announcements_connection",
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
      "storageKey": "__AdminAreaAnnouncementList_announcements_connection(sort:\"CREATED_AT_DESC\")"
    }
  ],
  "type": "Query",
  "abstractKey": null
};
})();

(node as any).hash = "fb43ced1dfdae5a000a22ea03f4e56be";

export default node;
