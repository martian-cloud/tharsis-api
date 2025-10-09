/**
 * @generated SignedSource<<2a8844ac669d7ca70da79f7f68e3a44a>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type HomeWorkspaceListPaginationQuery$variables = {
  after?: string | null | undefined;
  first?: number | null | undefined;
  search?: string | null | undefined;
};
export type HomeWorkspaceListPaginationQuery$data = {
  readonly " $fragmentSpreads": FragmentRefs<"HomeWorkspaceListFragment_workspaces">;
};
export type HomeWorkspaceListPaginationQuery = {
  response: HomeWorkspaceListPaginationQuery$data;
  variables: HomeWorkspaceListPaginationQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "after"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "first"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "search"
  }
],
v1 = [
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
    "kind": "Variable",
    "name": "search",
    "variableName": "search"
  },
  {
    "kind": "Literal",
    "name": "sort",
    "value": "FULL_PATH_ASC"
  }
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "HomeWorkspaceListPaginationQuery",
    "selections": [
      {
        "args": null,
        "kind": "FragmentSpread",
        "name": "HomeWorkspaceListFragment_workspaces"
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "HomeWorkspaceListPaginationQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "WorkspaceConnection",
        "kind": "LinkedField",
        "name": "workspaces",
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
                    "name": "name",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "fullPath",
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
        "args": (v1/*: any*/),
        "filters": [
          "search",
          "sort"
        ],
        "handle": "connection",
        "key": "HomeWorkspaceList_workspaces",
        "kind": "LinkedHandle",
        "name": "workspaces"
      }
    ]
  },
  "params": {
    "cacheID": "7b1babaf512bb6977b8746dab2901520",
    "id": null,
    "metadata": {},
    "name": "HomeWorkspaceListPaginationQuery",
    "operationKind": "query",
    "text": "query HomeWorkspaceListPaginationQuery(\n  $after: String\n  $first: Int\n  $search: String\n) {\n  ...HomeWorkspaceListFragment_workspaces\n}\n\nfragment HomeWorkspaceListFragment_workspaces on Query {\n  workspaces(first: $first, after: $after, search: $search, sort: FULL_PATH_ASC) {\n    totalCount\n    edges {\n      node {\n        id\n        ...HomeWorkspaceListItemFragment_workspace\n        __typename\n      }\n      cursor\n    }\n    pageInfo {\n      endCursor\n      hasNextPage\n    }\n  }\n}\n\nfragment HomeWorkspaceListItemFragment_workspace on Workspace {\n  name\n  fullPath\n}\n"
  }
};
})();

(node as any).hash = "f9d2d0f08007c5caa220d281fb0b9f42";

export default node;
