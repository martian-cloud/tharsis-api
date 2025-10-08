/**
 * @generated SignedSource<<46fe8b7b13622d97347cbba199006080>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type WorkspaceSearchQuery$variables = {
  after?: string | null | undefined;
  before?: string | null | undefined;
  first?: number | null | undefined;
  last?: number | null | undefined;
  search?: string | null | undefined;
};
export type WorkspaceSearchQuery$data = {
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceSearchFragment_workspaces">;
};
export type WorkspaceSearchQuery = {
  response: WorkspaceSearchQuery$data;
  variables: WorkspaceSearchQuery$variables;
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
  "name": "before"
},
v2 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "first"
},
v3 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "last"
},
v4 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "search"
},
v5 = [
  {
    "kind": "Variable",
    "name": "after",
    "variableName": "after"
  },
  {
    "kind": "Variable",
    "name": "before",
    "variableName": "before"
  },
  {
    "kind": "Variable",
    "name": "first",
    "variableName": "first"
  },
  {
    "kind": "Variable",
    "name": "last",
    "variableName": "last"
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
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/),
      (v2/*: any*/),
      (v3/*: any*/),
      (v4/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "WorkspaceSearchQuery",
    "selections": [
      {
        "args": null,
        "kind": "FragmentSpread",
        "name": "WorkspaceSearchFragment_workspaces"
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [
      (v2/*: any*/),
      (v3/*: any*/),
      (v0/*: any*/),
      (v1/*: any*/),
      (v4/*: any*/)
    ],
    "kind": "Operation",
    "name": "WorkspaceSearchQuery",
    "selections": [
      {
        "alias": null,
        "args": (v5/*: any*/),
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
                    "concreteType": "ResourceMetadata",
                    "kind": "LinkedField",
                    "name": "metadata",
                    "plural": false,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "updatedAt",
                        "storageKey": null
                      }
                    ],
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
                    "name": "description",
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
      {
        "alias": null,
        "args": (v5/*: any*/),
        "filters": [
          "search",
          "sort"
        ],
        "handle": "connection",
        "key": "WorkspaceSearch_workspaces",
        "kind": "LinkedHandle",
        "name": "workspaces"
      }
    ]
  },
  "params": {
    "cacheID": "69e123f5cb5d30451332bea70a46866d",
    "id": null,
    "metadata": {},
    "name": "WorkspaceSearchQuery",
    "operationKind": "query",
    "text": "query WorkspaceSearchQuery(\n  $first: Int\n  $last: Int\n  $after: String\n  $before: String\n  $search: String\n) {\n  ...WorkspaceSearchFragment_workspaces\n}\n\nfragment WorkspaceSearchFragment_workspaces on Query {\n  workspaces(after: $after, before: $before, first: $first, last: $last, search: $search, sort: FULL_PATH_ASC) {\n    totalCount\n    edges {\n      node {\n        id\n        ...WorkspaceSearchListItemFragment_workspace\n        __typename\n      }\n      cursor\n    }\n    pageInfo {\n      endCursor\n      hasNextPage\n      hasPreviousPage\n      startCursor\n    }\n  }\n}\n\nfragment WorkspaceSearchListItemFragment_workspace on Workspace {\n  metadata {\n    updatedAt\n  }\n  id\n  name\n  description\n  fullPath\n}\n"
  }
};
})();

(node as any).hash = "9c60184daed4fd738db3802242ed4c2d";

export default node;
