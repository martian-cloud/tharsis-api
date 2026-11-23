/**
 * @generated SignedSource<<87ee481ffc584aa3994f97e4a430c00a>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type VCSProviderWorkspaceListQuery$variables = {
  after?: string | null | undefined;
  first?: number | null | undefined;
  vcsProviderId: string;
};
export type VCSProviderWorkspaceListQuery$data = {
  readonly node: {
    readonly vcsProvider: {
      readonly " $fragmentSpreads": FragmentRefs<"VCSProviderWorkspaceListFragment_workspaces">;
    } | null | undefined;
  } | null | undefined;
};
export type VCSProviderWorkspaceListQuery = {
  response: VCSProviderWorkspaceListQuery$data;
  variables: VCSProviderWorkspaceListQuery$variables;
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
  "name": "vcsProviderId"
},
v3 = [
  {
    "kind": "Variable",
    "name": "id",
    "variableName": "vcsProviderId"
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
      (v0/*:: as any*/),
      (v1/*:: as any*/),
      (v2/*:: as any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "VCSProviderWorkspaceListQuery",
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
                {
                  "args": null,
                  "kind": "FragmentSpread",
                  "name": "VCSProviderWorkspaceListFragment_workspaces"
                }
              ],
              "type": "VCSProvider",
              "abstractKey": null
            },
            "kind": "AliasedInlineFragmentSpread",
            "name": "vcsProvider"
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
      (v1/*:: as any*/),
      (v0/*:: as any*/),
      (v2/*:: as any*/)
    ],
    "kind": "Operation",
    "name": "VCSProviderWorkspaceListQuery",
    "selections": [
      {
        "alias": null,
        "args": (v3/*:: as any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          (v4/*:: as any*/),
          (v5/*:: as any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              {
                "alias": null,
                "args": (v6/*:: as any*/),
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
                          (v5/*:: as any*/),
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
                          (v4/*:: as any*/)
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
                "args": (v6/*:: as any*/),
                "filters": [
                  "sort"
                ],
                "handle": "connection",
                "key": "VCSProviderWorkspaceList_workspaces",
                "kind": "LinkedHandle",
                "name": "workspaces"
              }
            ],
            "type": "VCSProvider",
            "abstractKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "e2cd52302f702175cf110464eecf22f6",
    "id": null,
    "metadata": {},
    "name": "VCSProviderWorkspaceListQuery",
    "operationKind": "query",
    "text": "query VCSProviderWorkspaceListQuery(\n  $first: Int\n  $after: String\n  $vcsProviderId: String!\n) {\n  node(id: $vcsProviderId) {\n    __typename\n    ... on VCSProvider {\n      ...VCSProviderWorkspaceListFragment_workspaces\n    }\n    id\n  }\n}\n\nfragment VCSProviderWorkspaceListFragment_workspaces on VCSProvider {\n  workspaces(after: $after, first: $first, sort: FULL_PATH_ASC) {\n    totalCount\n    edges {\n      node {\n        id\n        ...VCSProviderWorkspaceListItemFragment_workspace\n        __typename\n      }\n      cursor\n    }\n    pageInfo {\n      endCursor\n      hasNextPage\n    }\n  }\n  id\n}\n\nfragment VCSProviderWorkspaceListItemFragment_workspace on Workspace {\n  metadata {\n    updatedAt\n  }\n  name\n  description\n  fullPath\n}\n"
  }
};
})();

(node as any).hash = "06738815ef7bba24e8734fd319221fcf";

export default node;
