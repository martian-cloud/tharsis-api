/**
 * @generated SignedSource<<2c7403c748fb422b68c1b6d1efba3f64>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type NamespaceListPaginationQuery$variables = {
  after?: string | null | undefined;
  before?: string | null | undefined;
  first?: number | null | undefined;
  includeNested?: boolean | null | undefined;
  last?: number | null | undefined;
  parentId?: string | null | undefined;
  search?: string | null | undefined;
  topLevel?: boolean | null | undefined;
};
export type NamespaceListPaginationQuery$data = {
  readonly " $fragmentSpreads": FragmentRefs<"NamespaceListFragment_namespaces">;
};
export type NamespaceListPaginationQuery = {
  response: NamespaceListPaginationQuery$data;
  variables: NamespaceListPaginationQuery$variables;
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
    "name": "includeNested"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "last"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "parentId"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "search"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "topLevel"
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
    "name": "includeNested",
    "variableName": "includeNested"
  },
  {
    "kind": "Variable",
    "name": "last",
    "variableName": "last"
  },
  {
    "kind": "Variable",
    "name": "parentId",
    "variableName": "parentId"
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
  },
  {
    "kind": "Variable",
    "name": "topLevel",
    "variableName": "topLevel"
  }
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "NamespaceListPaginationQuery",
    "selections": [
      {
        "args": null,
        "kind": "FragmentSpread",
        "name": "NamespaceListFragment_namespaces"
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "NamespaceListPaginationQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "NamespaceConnection",
        "kind": "LinkedField",
        "name": "namespaces",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "NamespaceEdge",
            "kind": "LinkedField",
            "name": "edges",
            "plural": true,
            "selections": [
              {
                "alias": null,
                "args": null,
                "concreteType": null,
                "kind": "LinkedField",
                "name": "node",
                "plural": false,
                "selections": [
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
                    "kind": "ScalarField",
                    "name": "id",
                    "storageKey": null
                  },
                  {
                    "kind": "TypeDiscriminator",
                    "abstractKey": "__isNamespace"
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
                    "kind": "InlineFragment",
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "hasChildren",
                        "storageKey": null
                      }
                    ],
                    "type": "Group",
                    "abstractKey": null
                  },
                  {
                    "kind": "InlineFragment",
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "destroyed",
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "WorkspaceLabel",
                        "kind": "LinkedField",
                        "name": "labels",
                        "plural": true,
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "key",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "value",
                            "storageKey": null
                          }
                        ],
                        "storageKey": null
                      }
                    ],
                    "type": "Workspace",
                    "abstractKey": null
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
          "includeNested",
          "parentId",
          "search",
          "sort",
          "topLevel"
        ],
        "handle": "connection",
        "key": "NamespaceList_namespaces",
        "kind": "LinkedHandle",
        "name": "namespaces"
      }
    ]
  },
  "params": {
    "cacheID": "8bc3b274a27793a2530cfbd44b2d5c03",
    "id": null,
    "metadata": {},
    "name": "NamespaceListPaginationQuery",
    "operationKind": "query",
    "text": "query NamespaceListPaginationQuery(\n  $after: String\n  $before: String\n  $first: Int\n  $includeNested: Boolean\n  $last: Int\n  $parentId: String\n  $search: String\n  $topLevel: Boolean\n) {\n  ...NamespaceListFragment_namespaces\n}\n\nfragment NamespaceListFragment_namespaces on Query {\n  namespaces(after: $after, before: $before, first: $first, includeNested: $includeNested, last: $last, parentId: $parentId, search: $search, sort: FULL_PATH_ASC, topLevel: $topLevel) {\n    edges {\n      node {\n        __typename\n        id\n        ...NamespaceListItemFragment_namespace\n      }\n      cursor\n    }\n    pageInfo {\n      endCursor\n      hasNextPage\n      hasPreviousPage\n      startCursor\n    }\n  }\n}\n\nfragment NamespaceListItemFragment_namespace on Namespace {\n  __isNamespace: __typename\n  __typename\n  id\n  name\n  description\n  fullPath\n  ... on Group {\n    hasChildren\n  }\n  ... on Workspace {\n    destroyed\n    labels {\n      key\n      value\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "25b518af2407cb98d40f0e2d8f135c5d";

export default node;
