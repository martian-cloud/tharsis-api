/**
 * @generated SignedSource<<0252323fb42af5906c719529f794938b>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupListQuery$variables = {
  after?: string | null | undefined;
  before?: string | null | undefined;
  first?: number | null | undefined;
  last?: number | null | undefined;
  parentPath?: string | null | undefined;
  search?: string | null | undefined;
};
export type GroupListQuery$data = {
  readonly " $fragmentSpreads": FragmentRefs<"GroupListFragment_groups">;
};
export type GroupListQuery = {
  response: GroupListQuery$data;
  variables: GroupListQuery$variables;
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
v6 = [
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
    "kind": "Literal",
    "name": "sort",
    "value": "FULL_PATH_ASC"
  }
],
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "totalCount",
  "storageKey": null
},
v8 = [
  {
    "kind": "Literal",
    "name": "first",
    "value": 0
  }
],
v9 = [
  (v7/*: any*/)
];
return {
  "fragment": {
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/),
      (v2/*: any*/),
      (v3/*: any*/),
      (v4/*: any*/),
      (v5/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "GroupListQuery",
    "selections": [
      {
        "args": null,
        "kind": "FragmentSpread",
        "name": "GroupListFragment_groups"
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
      (v4/*: any*/),
      (v5/*: any*/)
    ],
    "kind": "Operation",
    "name": "GroupListQuery",
    "selections": [
      {
        "alias": null,
        "args": (v6/*: any*/),
        "concreteType": "GroupConnection",
        "kind": "LinkedField",
        "name": "groups",
        "plural": false,
        "selections": [
          (v7/*: any*/),
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
                    "args": (v8/*: any*/),
                    "concreteType": "GroupConnection",
                    "kind": "LinkedField",
                    "name": "descendentGroups",
                    "plural": false,
                    "selections": (v9/*: any*/),
                    "storageKey": "descendentGroups(first:0)"
                  },
                  {
                    "alias": null,
                    "args": (v8/*: any*/),
                    "concreteType": "WorkspaceConnection",
                    "kind": "LinkedField",
                    "name": "workspaces",
                    "plural": false,
                    "selections": (v9/*: any*/),
                    "storageKey": "workspaces(first:0)"
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
        "args": (v6/*: any*/),
        "filters": [
          "parentPath",
          "search",
          "sort"
        ],
        "handle": "connection",
        "key": "GroupList_groups",
        "kind": "LinkedHandle",
        "name": "groups"
      }
    ]
  },
  "params": {
    "cacheID": "ab1090155dc91fea807af2b5dd44cee9",
    "id": null,
    "metadata": {},
    "name": "GroupListQuery",
    "operationKind": "query",
    "text": "query GroupListQuery(\n  $first: Int\n  $last: Int\n  $after: String\n  $before: String\n  $parentPath: String\n  $search: String\n) {\n  ...GroupListFragment_groups\n}\n\nfragment GroupListFragment_groups on Query {\n  groups(after: $after, before: $before, first: $first, last: $last, parentPath: $parentPath, search: $search, sort: FULL_PATH_ASC) {\n    totalCount\n    edges {\n      node {\n        id\n        ...GroupListItemFragment_group\n        __typename\n      }\n      cursor\n    }\n    pageInfo {\n      endCursor\n      hasNextPage\n      hasPreviousPage\n      startCursor\n    }\n  }\n}\n\nfragment GroupListItemFragment_group on Group {\n  metadata {\n    updatedAt\n  }\n  id\n  name\n  description\n  fullPath\n  descendentGroups(first: 0) {\n    totalCount\n  }\n  workspaces(first: 0) {\n    totalCount\n  }\n}\n"
  }
};
})();

(node as any).hash = "9a0c8a7b05522bd0f0025dc8dcbb647c";

export default node;
