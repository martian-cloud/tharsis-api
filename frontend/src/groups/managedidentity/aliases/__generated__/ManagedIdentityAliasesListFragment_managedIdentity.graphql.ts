/**
 * @generated SignedSource<<2d59677811493f87c344328f72ca0ce4>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ManagedIdentityAliasesListFragment_managedIdentity$data = {
  readonly aliases: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly id: string;
        readonly " $fragmentSpreads": FragmentRefs<"ManagedIdentityAliasesListItemFragment_managedIdentity">;
      } | null | undefined;
    } | null | undefined> | null | undefined;
  };
  readonly id: string;
  readonly " $fragmentType": "ManagedIdentityAliasesListFragment_managedIdentity";
};
export type ManagedIdentityAliasesListFragment_managedIdentity$key = {
  readonly " $data"?: ManagedIdentityAliasesListFragment_managedIdentity$data;
  readonly " $fragmentSpreads": FragmentRefs<"ManagedIdentityAliasesListFragment_managedIdentity">;
};

import ManagedIdentityAliasesListPaginationQuery_graphql from './ManagedIdentityAliasesListPaginationQuery.graphql';

const node: ReaderFragment = (function(){
var v0 = [
  "aliases"
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
      "fragmentPathInResult": [
        "node"
      ],
      "operation": ManagedIdentityAliasesListPaginationQuery_graphql,
      "identifierInfo": {
        "identifierField": "id",
        "identifierQueryVariableName": "id"
      }
    }
  },
  "name": "ManagedIdentityAliasesListFragment_managedIdentity",
  "selections": [
    (v1/*: any*/),
    {
      "alias": "aliases",
      "args": null,
      "concreteType": "ManagedIdentityConnection",
      "kind": "LinkedField",
      "name": "__ManagedIdentityAliasesList_aliases_connection",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "concreteType": "ManagedIdentityEdge",
          "kind": "LinkedField",
          "name": "edges",
          "plural": true,
          "selections": [
            {
              "alias": null,
              "args": null,
              "concreteType": "ManagedIdentity",
              "kind": "LinkedField",
              "name": "node",
              "plural": false,
              "selections": [
                (v1/*: any*/),
                {
                  "args": null,
                  "kind": "FragmentSpread",
                  "name": "ManagedIdentityAliasesListItemFragment_managedIdentity"
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
      "storageKey": null
    }
  ],
  "type": "ManagedIdentity",
  "abstractKey": null
};
})();

(node as any).hash = "98009267d8d026fdad904d90830df5e9";

export default node;
