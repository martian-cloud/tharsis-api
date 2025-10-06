/**
 * @generated SignedSource<<d2f0967104ba300560b68b36ace6167d>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type RunnerType = "group" | "shared" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type AssignedServiceAccountListFragment_assignedServiceAccounts$data = {
  readonly assignedServiceAccounts: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly id: string;
        readonly " $fragmentSpreads": FragmentRefs<"AssignedServiceAccountListItemFragment_assignedServiceAccount">;
      } | null | undefined;
    } | null | undefined> | null | undefined;
    readonly totalCount: number;
  };
  readonly group: {
    readonly fullPath: string;
  };
  readonly id: string;
  readonly resourcePath: string;
  readonly type: RunnerType;
  readonly " $fragmentType": "AssignedServiceAccountListFragment_assignedServiceAccounts";
};
export type AssignedServiceAccountListFragment_assignedServiceAccounts$key = {
  readonly " $data"?: AssignedServiceAccountListFragment_assignedServiceAccounts$data;
  readonly " $fragmentSpreads": FragmentRefs<"AssignedServiceAccountListFragment_assignedServiceAccounts">;
};

import AssignedServiceAccountListPaginationQuery_graphql from './AssignedServiceAccountListPaginationQuery.graphql';

const node: ReaderFragment = (function(){
var v0 = [
  "assignedServiceAccounts"
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
      "operation": AssignedServiceAccountListPaginationQuery_graphql,
      "identifierInfo": {
        "identifierField": "id",
        "identifierQueryVariableName": "id"
      }
    }
  },
  "name": "AssignedServiceAccountListFragment_assignedServiceAccounts",
  "selections": [
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
      "name": "resourcePath",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "Group",
      "kind": "LinkedField",
      "name": "group",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "fullPath",
          "storageKey": null
        }
      ],
      "storageKey": null
    },
    {
      "alias": "assignedServiceAccounts",
      "args": null,
      "concreteType": "ServiceAccountConnection",
      "kind": "LinkedField",
      "name": "__AssignedServiceAccountList_assignedServiceAccounts_connection",
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
          "concreteType": "ServiceAccountEdge",
          "kind": "LinkedField",
          "name": "edges",
          "plural": true,
          "selections": [
            {
              "alias": null,
              "args": null,
              "concreteType": "ServiceAccount",
              "kind": "LinkedField",
              "name": "node",
              "plural": false,
              "selections": [
                (v1/*: any*/),
                {
                  "args": null,
                  "kind": "FragmentSpread",
                  "name": "AssignedServiceAccountListItemFragment_assignedServiceAccount"
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
  "type": "Runner",
  "abstractKey": null
};
})();

(node as any).hash = "4162eed734978b845c8eec9ff230b714";

export default node;
