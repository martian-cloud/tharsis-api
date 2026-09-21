/**
 * @generated SignedSource<<2ed1925d6e6266e038825edd6aeeadfc>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type AdminAreaEmailRecipientListFragment_outbox$data = {
  readonly id: string;
  readonly recipients: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly id: string;
        readonly " $fragmentSpreads": FragmentRefs<"AdminAreaEmailRecipientListItemFragment_recipient">;
      } | null | undefined;
    } | null | undefined> | null | undefined;
  };
  readonly " $fragmentType": "AdminAreaEmailRecipientListFragment_outbox";
};
export type AdminAreaEmailRecipientListFragment_outbox$key = {
  readonly " $data"?: AdminAreaEmailRecipientListFragment_outbox$data;
  readonly " $fragmentSpreads": FragmentRefs<"AdminAreaEmailRecipientListFragment_outbox">;
};

import AdminAreaEmailRecipientListPaginationQuery_graphql from './AdminAreaEmailRecipientListPaginationQuery.graphql';

const node: ReaderFragment = (function(){
var v0 = [
  "recipients"
],
v1 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
};
return {
  "argumentDefinitions": [
    {
      "kind": "RootArgument",
      "name": "after"
    },
    {
      "kind": "RootArgument",
      "name": "deliveryStatuses"
    },
    {
      "kind": "RootArgument",
      "name": "first"
    },
    {
      "kind": "RootArgument",
      "name": "hasIssues"
    },
    {
      "kind": "RootArgument",
      "name": "hasOpened"
    },
    {
      "kind": "RootArgument",
      "name": "search"
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
      "fragmentPathInResult": [
        "node"
      ],
      "operation": AdminAreaEmailRecipientListPaginationQuery_graphql,
      "identifierInfo": {
        "identifierField": "id",
        "identifierQueryVariableName": "id"
      }
    }
  },
  "name": "AdminAreaEmailRecipientListFragment_outbox",
  "selections": [
    {
      "alias": "recipients",
      "args": [
        {
          "kind": "Variable",
          "name": "deliveryStatuses",
          "variableName": "deliveryStatuses"
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
      ],
      "concreteType": "EmailRecipientConnection",
      "kind": "LinkedField",
      "name": "__AdminAreaEmailRecipientList_recipients_connection",
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
                (v1/*: any*/),
                {
                  "args": null,
                  "kind": "FragmentSpread",
                  "name": "AdminAreaEmailRecipientListItemFragment_recipient"
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
    (v1/*: any*/)
  ],
  "type": "EmailOutboxItem",
  "abstractKey": null
};
})();

(node as any).hash = "c1dcd3238fb8c759568989536b4cab44";

export default node;
