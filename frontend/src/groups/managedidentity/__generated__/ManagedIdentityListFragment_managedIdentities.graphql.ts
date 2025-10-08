/**
 * @generated SignedSource<<b710bed585de6fd8fd3e821221783020>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ManagedIdentityListFragment_managedIdentities$data = {
  readonly id: string;
  readonly managedIdentities: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly groupPath: string;
        readonly id: string;
        readonly " $fragmentSpreads": FragmentRefs<"ManagedIdentityListItemFragment_managedIdentity">;
      } | null | undefined;
    } | null | undefined> | null | undefined;
    readonly totalCount: number;
  };
  readonly " $fragmentType": "ManagedIdentityListFragment_managedIdentities";
};
export type ManagedIdentityListFragment_managedIdentities$key = {
  readonly " $data"?: ManagedIdentityListFragment_managedIdentities$data;
  readonly " $fragmentSpreads": FragmentRefs<"ManagedIdentityListFragment_managedIdentities">;
};

import ManagedIdentityListPaginationQuery_graphql from './ManagedIdentityListPaginationQuery.graphql';

const node: ReaderFragment = (function(){
var v0 = [
  "managedIdentities"
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
      "operation": ManagedIdentityListPaginationQuery_graphql,
      "identifierInfo": {
        "identifierField": "id",
        "identifierQueryVariableName": "id"
      }
    }
  },
  "name": "ManagedIdentityListFragment_managedIdentities",
  "selections": [
    {
      "alias": "managedIdentities",
      "args": [
        {
          "kind": "Literal",
          "name": "includeInherited",
          "value": true
        },
        {
          "kind": "Variable",
          "name": "search",
          "variableName": "search"
        },
        {
          "kind": "Literal",
          "name": "sort",
          "value": "GROUP_LEVEL_DESC"
        }
      ],
      "concreteType": "ManagedIdentityConnection",
      "kind": "LinkedField",
      "name": "__ManagedIdentityList_managedIdentities_connection",
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
                  "alias": null,
                  "args": null,
                  "kind": "ScalarField",
                  "name": "groupPath",
                  "storageKey": null
                },
                {
                  "args": null,
                  "kind": "FragmentSpread",
                  "name": "ManagedIdentityListItemFragment_managedIdentity"
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
    },
    (v1/*: any*/)
  ],
  "type": "Group",
  "abstractKey": null
};
})();

(node as any).hash = "be9554f770c64d06a4f067041ba1a56e";

export default node;
