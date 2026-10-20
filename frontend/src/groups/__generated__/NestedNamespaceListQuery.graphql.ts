/**
 * @generated SignedSource<<17e97b872cb75fb0869de1d8ab83f6ed>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type NestedNamespaceListQuery$variables = {
  after?: string | null | undefined;
  before?: string | null | undefined;
  first?: number | null | undefined;
  last?: number | null | undefined;
  parentId: string;
};
export type NestedNamespaceListQuery$data = {
  readonly " $fragmentSpreads": FragmentRefs<"NestedNamespaceListFragment_namespaces">;
};
export type NestedNamespaceListQuery = {
  response: NestedNamespaceListQuery$data;
  variables: NestedNamespaceListQuery$variables;
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
  "name": "parentId"
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
    "name": "parentId",
    "variableName": "parentId"
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
    "name": "NestedNamespaceListQuery",
    "selections": [
      {
        "args": null,
        "kind": "FragmentSpread",
        "name": "NestedNamespaceListFragment_namespaces"
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
    "name": "NestedNamespaceListQuery",
    "selections": [
      {
        "alias": null,
        "args": (v5/*: any*/),
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
        "args": (v5/*: any*/),
        "filters": [
          "parentId",
          "sort"
        ],
        "handle": "connection",
        "key": "NestedNamespaceList_namespaces",
        "kind": "LinkedHandle",
        "name": "namespaces"
      }
    ]
  },
  "params": {
    "cacheID": "be418c7292e4a9b61c13c37fd1e066e9",
    "id": null,
    "metadata": {},
    "name": "NestedNamespaceListQuery",
    "operationKind": "query",
    "text": "query NestedNamespaceListQuery(\n  $first: Int\n  $last: Int\n  $after: String\n  $before: String\n  $parentId: String!\n) {\n  ...NestedNamespaceListFragment_namespaces\n}\n\nfragment NamespaceListItemFragment_namespace on Namespace {\n  __isNamespace: __typename\n  __typename\n  id\n  name\n  description\n  fullPath\n  ... on Group {\n    hasChildren\n  }\n  ... on Workspace {\n    destroyed\n    labels {\n      key\n      value\n    }\n  }\n}\n\nfragment NestedNamespaceListFragment_namespaces on Query {\n  namespaces(after: $after, before: $before, first: $first, last: $last, parentId: $parentId, sort: FULL_PATH_ASC) {\n    edges {\n      node {\n        __typename\n        id\n        ...NamespaceListItemFragment_namespace\n      }\n      cursor\n    }\n    pageInfo {\n      endCursor\n      hasNextPage\n      hasPreviousPage\n      startCursor\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "56f5eb63961a237023ba7477335a0b74";

export default node;
