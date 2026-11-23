/**
 * @generated SignedSource<<b82527fdb6c852d6b9bd374607384f61>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type VCSProviderWorkspaceListFragment_workspaces$data = {
  readonly id: string;
  readonly workspaces: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly id: string;
        readonly " $fragmentSpreads": FragmentRefs<"VCSProviderWorkspaceListItemFragment_workspace">;
      } | null | undefined;
    } | null | undefined> | null | undefined;
    readonly totalCount: number;
  };
  readonly " $fragmentType": "VCSProviderWorkspaceListFragment_workspaces";
};
export type VCSProviderWorkspaceListFragment_workspaces$key = {
  readonly " $data"?: VCSProviderWorkspaceListFragment_workspaces$data;
  readonly " $fragmentSpreads": FragmentRefs<"VCSProviderWorkspaceListFragment_workspaces">;
};

import VCSProviderWorkspaceListPaginationQuery_graphql from './VCSProviderWorkspaceListPaginationQuery.graphql';

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
      "fragmentPathInResult": [
        "node"
      ],
      "operation": VCSProviderWorkspaceListPaginationQuery_graphql,
      "identifierInfo": {
        "identifierField": "id",
        "identifierQueryVariableName": "id"
      }
    }
  },
  "name": "VCSProviderWorkspaceListFragment_workspaces",
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
      "name": "__VCSProviderWorkspaceList_workspaces_connection",
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
                (v1/*:: as any*/),
                {
                  "args": null,
                  "kind": "FragmentSpread",
                  "name": "VCSProviderWorkspaceListItemFragment_workspace"
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
      "storageKey": "__VCSProviderWorkspaceList_workspaces_connection(sort:\"FULL_PATH_ASC\")"
    },
    (v1/*:: as any*/)
  ],
  "type": "VCSProvider",
  "abstractKey": null
};
})();

(node as any).hash = "3ffe85eef56860c93ecdc388abc3d680";

export default node;
