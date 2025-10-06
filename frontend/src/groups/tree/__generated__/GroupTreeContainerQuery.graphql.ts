/**
 * @generated SignedSource<<8e919d1cbe08efaeacfbe873e0ed9146>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupSort = "FULL_PATH_ASC" | "FULL_PATH_DESC" | "GROUP_LEVEL_ASC" | "GROUP_LEVEL_DESC" | "UPDATED_AT_ASC" | "UPDATED_AT_DESC" | "%future added value";
export type GroupTreeContainerQuery$variables = {
  after?: string | null | undefined;
  before?: string | null | undefined;
  first?: number | null | undefined;
  last?: number | null | undefined;
  parentPath?: string | null | undefined;
  search?: string | null | undefined;
  sort?: GroupSort | null | undefined;
};
export type GroupTreeContainerQuery$data = {
  readonly " $fragmentSpreads": FragmentRefs<"GroupTreeContainerFragment_groups" | "GroupTreeContainerFragment_me">;
};
export type GroupTreeContainerQuery = {
  response: GroupTreeContainerQuery$data;
  variables: GroupTreeContainerQuery$variables;
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
  "name": "parentPath"
},
v5 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "search"
},
v6 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "sort"
},
v7 = [
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
v8 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "totalCount",
  "storageKey": null
},
v9 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v10 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "__typename",
  "storageKey": null
},
v11 = [
  {
    "kind": "Literal",
    "name": "first",
    "value": 0
  }
],
v12 = [
  (v8/*: any*/)
];
return {
  "fragment": {
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/),
      (v2/*: any*/),
      (v3/*: any*/),
      (v4/*: any*/),
      (v5/*: any*/),
      (v6/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "GroupTreeContainerQuery",
    "selections": [
      {
        "args": null,
        "kind": "FragmentSpread",
        "name": "GroupTreeContainerFragment_groups"
      },
      {
        "args": null,
        "kind": "FragmentSpread",
        "name": "GroupTreeContainerFragment_me"
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
      (v5/*: any*/),
      (v4/*: any*/),
      (v6/*: any*/)
    ],
    "kind": "Operation",
    "name": "GroupTreeContainerQuery",
    "selections": [
      {
        "alias": null,
        "args": (v7/*: any*/),
        "concreteType": "GroupConnection",
        "kind": "LinkedField",
        "name": "groups",
        "plural": false,
        "selections": [
          (v8/*: any*/),
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
                  (v9/*: any*/),
                  (v10/*: any*/),
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
                    "args": (v11/*: any*/),
                    "concreteType": "GroupConnection",
                    "kind": "LinkedField",
                    "name": "descendentGroups",
                    "plural": false,
                    "selections": (v12/*: any*/),
                    "storageKey": "descendentGroups(first:0)"
                  },
                  {
                    "alias": null,
                    "args": (v11/*: any*/),
                    "concreteType": "WorkspaceConnection",
                    "kind": "LinkedField",
                    "name": "workspaces",
                    "plural": false,
                    "selections": (v12/*: any*/),
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
        "args": (v7/*: any*/),
        "filters": [
          "search",
          "parentPath",
          "sort"
        ],
        "handle": "connection",
        "key": "GroupTreeContainer_groups",
        "kind": "LinkedHandle",
        "name": "groups"
      },
      {
        "alias": null,
        "args": null,
        "concreteType": null,
        "kind": "LinkedField",
        "name": "me",
        "plural": false,
        "selections": [
          (v10/*: any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "admin",
                "storageKey": null
              }
            ],
            "type": "User",
            "abstractKey": null
          },
          {
            "kind": "InlineFragment",
            "selections": [
              (v9/*: any*/)
            ],
            "type": "Node",
            "abstractKey": "__isNode"
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "df51c694528afb0a11a04e1eda753847",
    "id": null,
    "metadata": {},
    "name": "GroupTreeContainerQuery",
    "operationKind": "query",
    "text": "query GroupTreeContainerQuery(\n  $first: Int\n  $last: Int\n  $after: String\n  $before: String\n  $search: String\n  $parentPath: String\n  $sort: GroupSort\n) {\n  ...GroupTreeContainerFragment_groups\n  ...GroupTreeContainerFragment_me\n}\n\nfragment GroupTreeContainerFragment_groups on Query {\n  groups(after: $after, before: $before, first: $first, last: $last, search: $search, parentPath: $parentPath, sort: $sort) {\n    totalCount\n    edges {\n      node {\n        id\n        __typename\n      }\n      cursor\n    }\n    ...GroupTreeFragment_connection\n    pageInfo {\n      endCursor\n      hasNextPage\n      hasPreviousPage\n      startCursor\n    }\n  }\n}\n\nfragment GroupTreeContainerFragment_me on Query {\n  me {\n    __typename\n    ... on User {\n      admin\n    }\n    ... on Node {\n      __isNode: __typename\n      id\n    }\n  }\n}\n\nfragment GroupTreeFragment_connection on GroupConnection {\n  totalCount\n  edges {\n    node {\n      id\n      ...GroupTreeListItemFragment_group\n    }\n  }\n}\n\nfragment GroupTreeListItemFragment_group on Group {\n  metadata {\n    updatedAt\n  }\n  id\n  name\n  description\n  fullPath\n  descendentGroups(first: 0) {\n    totalCount\n  }\n  workspaces(first: 0) {\n    totalCount\n  }\n}\n"
  }
};
})();

(node as any).hash = "013da9326c978ed070205b5e0ebdb3cd";

export default node;
