/**
 * @generated SignedSource<<e338706a2adfedc2e926bb8242d78514>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ManagedIdentityWorkspaceListFragment_workspaces$data = {
  readonly id: string;
  readonly workspaces: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly id: string;
        readonly " $fragmentSpreads": FragmentRefs<"ManagedIdentityWorkspaceListItemFragment_workspace">;
      } | null | undefined;
    } | null | undefined> | null | undefined;
    readonly totalCount: number;
  };
  readonly " $fragmentType": "ManagedIdentityWorkspaceListFragment_workspaces";
};
export type ManagedIdentityWorkspaceListFragment_workspaces$key = {
  readonly " $data"?: ManagedIdentityWorkspaceListFragment_workspaces$data;
  readonly " $fragmentSpreads": FragmentRefs<"ManagedIdentityWorkspaceListFragment_workspaces">;
};

import ManagedIdentityWorkspaceListPaginationQuery_graphql from './ManagedIdentityWorkspaceListPaginationQuery.graphql';

const node: ReaderFragment = (function(){
var v0 = [
  "workspaces"
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
      "operation": ManagedIdentityWorkspaceListPaginationQuery_graphql,
      "identifierInfo": {
        "identifierField": "id",
        "identifierQueryVariableName": "id"
      }
    }
  },
  "name": "ManagedIdentityWorkspaceListFragment_workspaces",
  "selections": [
    {
      "alias": "workspaces",
      "args": [
        {
          "kind": "Literal",
          "name": "sort",
          "value": "FULL_PATH_ASC"
        }
      ],
      "concreteType": "WorkspaceConnection",
      "kind": "LinkedField",
      "name": "__ManagedIdentityWorkspaceList_workspaces_connection",
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
          "concreteType": "WorkspaceEdge",
          "kind": "LinkedField",
          "name": "edges",
          "plural": true,
          "selections": [
            {
              "alias": null,
              "args": null,
              "concreteType": "Workspace",
              "kind": "LinkedField",
              "name": "node",
              "plural": false,
              "selections": [
                (v1/*: any*/),
                {
                  "args": null,
                  "kind": "FragmentSpread",
                  "name": "ManagedIdentityWorkspaceListItemFragment_workspace"
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
      "storageKey": "__ManagedIdentityWorkspaceList_workspaces_connection(sort:\"FULL_PATH_ASC\")"
    },
    (v1/*: any*/)
  ],
  "type": "ManagedIdentity",
  "abstractKey": null
};
})();

(node as any).hash = "6c545d8ec902d78ba1cfa03b61e4dc6b";

export default node;
