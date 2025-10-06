/**
 * @generated SignedSource<<ae60cc794f825d0336e1f9b4c2f87b34>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupSort = "FULL_PATH_ASC" | "FULL_PATH_DESC" | "GROUP_LEVEL_ASC" | "GROUP_LEVEL_DESC" | "UPDATED_AT_ASC" | "UPDATED_AT_DESC" | "%future added value";
export type GroupsPaginationQuery$variables = {
  after?: string | null | undefined;
  before?: string | null | undefined;
  first?: number | null | undefined;
  last?: number | null | undefined;
  parentPath?: string | null | undefined;
  search?: string | null | undefined;
  sort?: GroupSort | null | undefined;
};
export type GroupsPaginationQuery$data = {
  readonly " $fragmentSpreads": FragmentRefs<"GroupTreeContainerFragment_groups">;
};
export type GroupsPaginationQuery = {
  response: GroupsPaginationQuery$data;
  variables: GroupsPaginationQuery$variables;
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
    "name": "before"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "first"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "last"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "parentPath"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "search"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "sort"
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
    "name": "parentPath",
    "variableName": "parentPath"
  },
  {
    "kind": "Variable",
    "name": "search",
    "variableName": "search"
  },
  {
    "kind": "Variable",
    "name": "sort",
    "variableName": "sort"
  }
],
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "totalCount",
  "storageKey": null
},
v3 = [
  {
    "kind": "Literal",
    "name": "first",
    "value": 0
  }
],
v4 = [
  (v2/*: any*/)
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "GroupsPaginationQuery",
    "selections": [
      {
        "args": null,
        "kind": "FragmentSpread",
        "name": "GroupTreeContainerFragment_groups"
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "GroupsPaginationQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "GroupConnection",
        "kind": "LinkedField",
        "name": "groups",
        "plural": false,
        "selections": [
          (v2/*: any*/),
          {
            "alias": null,
            "args": null,
            "concreteType": "GroupEdge",
            "kind": "LinkedField",
            "name": "edges",
            "plural": true,
            "selections": [
              {
                "alias": null,
                "args": null,
                "concreteType": "Group",
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
                    "name": "__typename",
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
                    "args": (v3/*: any*/),
                    "concreteType": "GroupConnection",
                    "kind": "LinkedField",
                    "name": "descendentGroups",
                    "plural": false,
                    "selections": (v4/*: any*/),
                    "storageKey": "descendentGroups(first:0)"
                  },
                  {
                    "alias": null,
                    "args": (v3/*: any*/),
                    "concreteType": "WorkspaceConnection",
                    "kind": "LinkedField",
                    "name": "workspaces",
                    "plural": false,
                    "selections": (v4/*: any*/),
                    "storageKey": "workspaces(first:0)"
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
        "args": (v1/*: any*/),
        "filters": [
          "search",
          "parentPath",
          "sort"
        ],
        "handle": "connection",
        "key": "GroupTreeContainer_groups",
        "kind": "LinkedHandle",
        "name": "groups"
      }
    ]
  },
  "params": {
    "cacheID": "035e36356ea2859f7c3053d407270a67",
    "id": null,
    "metadata": {},
    "name": "GroupsPaginationQuery",
    "operationKind": "query",
    "text": "query GroupsPaginationQuery(\n  $after: String\n  $before: String\n  $first: Int\n  $last: Int\n  $parentPath: String\n  $search: String\n  $sort: GroupSort\n) {\n  ...GroupTreeContainerFragment_groups\n}\n\nfragment GroupTreeContainerFragment_groups on Query {\n  groups(after: $after, before: $before, first: $first, last: $last, search: $search, parentPath: $parentPath, sort: $sort) {\n    totalCount\n    edges {\n      node {\n        id\n        __typename\n      }\n      cursor\n    }\n    ...GroupTreeFragment_connection\n    pageInfo {\n      endCursor\n      hasNextPage\n      hasPreviousPage\n      startCursor\n    }\n  }\n}\n\nfragment GroupTreeFragment_connection on GroupConnection {\n  totalCount\n  edges {\n    node {\n      id\n      ...GroupTreeListItemFragment_group\n    }\n  }\n}\n\nfragment GroupTreeListItemFragment_group on Group {\n  metadata {\n    updatedAt\n  }\n  id\n  name\n  description\n  fullPath\n  descendentGroups(first: 0) {\n    totalCount\n  }\n  workspaces(first: 0) {\n    totalCount\n  }\n}\n"
  }
};
})();

(node as any).hash = "0e182ad85e2a571871b3807f22988048";

export default node;
