/**
 * @generated SignedSource<<f69c9c8c9e14c24fc200744df6acc631>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ManagedIdentityWorkspaceListQuery$variables = {
  after?: string | null | undefined;
  first?: number | null | undefined;
  managedIdentityId: string;
};
export type ManagedIdentityWorkspaceListQuery$data = {
  readonly node: {
    readonly " $fragmentSpreads": FragmentRefs<"ManagedIdentityWorkspaceListFragment_workspaces">;
  } | null | undefined;
};
export type ManagedIdentityWorkspaceListQuery = {
  response: ManagedIdentityWorkspaceListQuery$data;
  variables: ManagedIdentityWorkspaceListQuery$variables;
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
  "name": "managedIdentityId"
},
v3 = [
  {
    "kind": "Variable",
    "name": "id",
    "variableName": "managedIdentityId"
  }
],
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "__typename",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v6 = [
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
      (v2/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "ManagedIdentityWorkspaceListQuery",
    "selections": [
      {
        "alias": null,
        "args": (v3/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          {
            "kind": "InlineFragment",
            "selections": [
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "ManagedIdentityWorkspaceListFragment_workspaces"
              }
            ],
            "type": "ManagedIdentity",
            "abstractKey": null
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
      (v1/*: any*/),
      (v0/*: any*/),
      (v2/*: any*/)
    ],
    "kind": "Operation",
    "name": "ManagedIdentityWorkspaceListQuery",
    "selections": [
      {
        "alias": null,
        "args": (v3/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          (v4/*: any*/),
          (v5/*: any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              {
                "alias": null,
                "args": (v6/*: any*/),
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
                          (v5/*: any*/),
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
                          (v4/*: any*/)
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
                "args": (v6/*: any*/),
                "filters": [
                  "sort"
                ],
                "handle": "connection",
                "key": "ManagedIdentityWorkspaceList_workspaces",
                "kind": "LinkedHandle",
                "name": "workspaces"
              }
            ],
            "type": "ManagedIdentity",
            "abstractKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "3c01fa58d779db679865751863ef4cff",
    "id": null,
    "metadata": {},
    "name": "ManagedIdentityWorkspaceListQuery",
    "operationKind": "query",
    "text": "query ManagedIdentityWorkspaceListQuery(\n  $first: Int\n  $after: String\n  $managedIdentityId: String!\n) {\n  node(id: $managedIdentityId) {\n    __typename\n    ... on ManagedIdentity {\n      ...ManagedIdentityWorkspaceListFragment_workspaces\n    }\n    id\n  }\n}\n\nfragment ManagedIdentityWorkspaceListFragment_workspaces on ManagedIdentity {\n  workspaces(after: $after, first: $first, sort: FULL_PATH_ASC) {\n    totalCount\n    edges {\n      node {\n        id\n        ...ManagedIdentityWorkspaceListItemFragment_workspace\n        __typename\n      }\n      cursor\n    }\n    pageInfo {\n      endCursor\n      hasNextPage\n    }\n  }\n  id\n}\n\nfragment ManagedIdentityWorkspaceListItemFragment_workspace on Workspace {\n  metadata {\n    updatedAt\n  }\n  id\n  name\n  description\n  fullPath\n}\n"
  }
};
})();

(node as any).hash = "fde456a974fdef6c80282e26b319a620";

export default node;
