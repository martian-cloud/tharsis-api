/**
 * @generated SignedSource<<f82946310f50be66447242f71c74e540>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type AssignedServiceAccountListQuery$variables = {
  after?: string | null | undefined;
  first: number;
  id: string;
};
export type AssignedServiceAccountListQuery$data = {
  readonly node: {
    readonly runner: {
      readonly id: string;
      readonly " $fragmentSpreads": FragmentRefs<"AssignedServiceAccountListFragment_assignedServiceAccounts">;
    } | null | undefined;
  } | null | undefined;
};
export type AssignedServiceAccountListQuery = {
  response: AssignedServiceAccountListQuery$data;
  variables: AssignedServiceAccountListQuery$variables;
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
  "name": "first"
},
v2 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "id"
},
v3 = [
  {
    "kind": "Variable",
    "name": "id",
    "variableName": "id"
  }
],
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "__typename",
  "storageKey": null
},
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "resourcePath",
  "storageKey": null
},
v7 = [
  {
    "kind": "Variable",
    "name": "after",
    "variableName": "after"
  },
  {
    "kind": "Variable",
    "name": "first",
    "variableName": "first"
  }
];
return {
  "fragment": {
    "argumentDefinitions": [
      (v0/*:: as any*/),
      (v1/*:: as any*/),
      (v2/*:: as any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "AssignedServiceAccountListQuery",
    "selections": [
      {
        "alias": null,
        "args": (v3/*:: as any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          {
            "fragment": {
              "kind": "InlineFragment",
              "selections": [
                (v4/*:: as any*/),
                {
                  "args": null,
                  "kind": "FragmentSpread",
                  "name": "AssignedServiceAccountListFragment_assignedServiceAccounts"
                }
              ],
              "type": "Runner",
              "abstractKey": null
            },
            "kind": "AliasedInlineFragmentSpread",
            "name": "runner"
          }
        ],
        "storageKey": null
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [
      (v2/*:: as any*/),
      (v1/*:: as any*/),
      (v0/*:: as any*/)
    ],
    "kind": "Operation",
    "name": "AssignedServiceAccountListQuery",
    "selections": [
      {
        "alias": null,
        "args": (v3/*:: as any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          (v5/*:: as any*/),
          (v4/*:: as any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "type",
                "storageKey": null
              },
              (v6/*:: as any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "Group",
                "kind": "LinkedField",
                "name": "group",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "fullPath",
                    "storageKey": null
                  },
                  (v4/*:: as any*/)
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": (v7/*:: as any*/),
                "concreteType": "ServiceAccountConnection",
                "kind": "LinkedField",
                "name": "assignedServiceAccounts",
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
                    "concreteType": "ServiceAccountEdge",
                    "kind": "LinkedField",
                    "name": "edges",
                    "plural": true,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "ServiceAccount",
                        "kind": "LinkedField",
                        "name": "node",
                        "plural": false,
                        "selections": [
                          (v4/*:: as any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "name",
                            "storageKey": null
                          },
                          (v6/*:: as any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "groupPath",
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
                          (v5/*:: as any*/)
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
                "args": (v7/*:: as any*/),
                "filters": null,
                "handle": "connection",
                "key": "AssignedServiceAccountList_assignedServiceAccounts",
                "kind": "LinkedHandle",
                "name": "assignedServiceAccounts"
              }
            ],
            "type": "Runner",
            "abstractKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "065a63b224f7873e00ada00529fe2689",
    "id": null,
    "metadata": {},
    "name": "AssignedServiceAccountListQuery",
    "operationKind": "query",
    "text": "query AssignedServiceAccountListQuery(\n  $id: String!\n  $first: Int!\n  $after: String\n) {\n  node(id: $id) {\n    __typename\n    ... on Runner {\n      id\n      ...AssignedServiceAccountListFragment_assignedServiceAccounts\n    }\n    id\n  }\n}\n\nfragment AssignedServiceAccountListFragment_assignedServiceAccounts on Runner {\n  type\n  resourcePath\n  group {\n    fullPath\n    id\n  }\n  assignedServiceAccounts(first: $first, after: $after) {\n    totalCount\n    edges {\n      node {\n        id\n        ...AssignedServiceAccountListItemFragment_assignedServiceAccount\n        __typename\n      }\n      cursor\n    }\n    pageInfo {\n      endCursor\n      hasNextPage\n    }\n  }\n  id\n}\n\nfragment AssignedServiceAccountListItemFragment_assignedServiceAccount on ServiceAccount {\n  id\n  name\n  resourcePath\n  groupPath\n  description\n  metadata {\n    updatedAt\n  }\n}\n"
  }
};
})();

(node as any).hash = "ad0eaefa5891b61ecaa458415d4cb51d";

export default node;
