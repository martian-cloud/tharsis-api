/**
 * @generated SignedSource<<3b2e8961585957609b0f855dec5e6e96>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupPackageListQuery$variables = {
  after?: string | null | undefined;
  before?: string | null | undefined;
  first?: number | null | undefined;
  groupId: string;
  last?: number | null | undefined;
  search?: string | null | undefined;
};
export type GroupPackageListQuery$data = {
  readonly node: {
    readonly " $fragmentSpreads": FragmentRefs<"GroupPackageListFragment_packages">;
  } | null | undefined;
};
export type GroupPackageListQuery = {
  response: GroupPackageListQuery$data;
  variables: GroupPackageListQuery$variables;
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
  "name": "groupId"
},
v4 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "last"
},
v5 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "search"
},
v6 = [
  {
    "kind": "Variable",
    "name": "id",
    "variableName": "groupId"
  }
],
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "__typename",
  "storageKey": null
},
v8 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v9 = {
  "kind": "Literal",
  "name": "includeInherited",
  "value": true
},
v10 = [
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
  (v9/*: any*/),
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
    "value": "GROUP_LEVEL_DESC"
  }
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
    "name": "GroupPackageListQuery",
    "selections": [
      {
        "alias": null,
        "args": (v6/*: any*/),
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
                "name": "GroupPackageListFragment_packages"
              }
            ],
            "type": "Group",
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
      (v2/*: any*/),
      (v4/*: any*/),
      (v0/*: any*/),
      (v1/*: any*/),
      (v3/*: any*/),
      (v5/*: any*/)
    ],
    "kind": "Operation",
    "name": "GroupPackageListQuery",
    "selections": [
      {
        "alias": null,
        "args": (v6/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          (v7/*: any*/),
          (v8/*: any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              {
                "alias": "allPackages",
                "args": [
                  {
                    "kind": "Literal",
                    "name": "first",
                    "value": 0
                  },
                  (v9/*: any*/)
                ],
                "concreteType": "PackageConnection",
                "kind": "LinkedField",
                "name": "packages",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "totalCount",
                    "storageKey": null
                  }
                ],
                "storageKey": "packages(first:0,includeInherited:true)"
              },
              {
                "alias": null,
                "args": (v10/*: any*/),
                "concreteType": "PackageConnection",
                "kind": "LinkedField",
                "name": "packages",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "PackageEdge",
                    "kind": "LinkedField",
                    "name": "edges",
                    "plural": true,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "Package",
                        "kind": "LinkedField",
                        "name": "node",
                        "plural": false,
                        "selections": [
                          (v8/*: any*/),
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
                            "name": "kind",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "visibility",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": "PackageVersion",
                            "kind": "LinkedField",
                            "name": "latestVersion",
                            "plural": false,
                            "selections": [
                              {
                                "alias": null,
                                "args": null,
                                "kind": "ScalarField",
                                "name": "version",
                                "storageKey": null
                              },
                              (v8/*: any*/)
                            ],
                            "storageKey": null
                          },
                          (v7/*: any*/)
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
                  },
                  {
                    "kind": "ClientExtension",
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "__id",
                        "storageKey": null
                      }
                    ]
                  }
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": (v10/*: any*/),
                "filters": [
                  "search",
                  "sort",
                  "includeInherited"
                ],
                "handle": "connection",
                "key": "PackageList_packages",
                "kind": "LinkedHandle",
                "name": "packages"
              }
            ],
            "type": "Group",
            "abstractKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "26b98a70b543b5917a762823de860271",
    "id": null,
    "metadata": {},
    "name": "GroupPackageListQuery",
    "operationKind": "query",
    "text": "query GroupPackageListQuery(\n  $first: Int\n  $last: Int\n  $after: String\n  $before: String\n  $groupId: String!\n  $search: String\n) {\n  node(id: $groupId) {\n    __typename\n    ... on Group {\n      ...GroupPackageListFragment_packages\n    }\n    id\n  }\n}\n\nfragment GroupPackageListFragment_packages on Group {\n  allPackages: packages(first: 0, includeInherited: true) {\n    totalCount\n  }\n  packages(after: $after, before: $before, first: $first, last: $last, search: $search, sort: GROUP_LEVEL_DESC, includeInherited: true) {\n    edges {\n      node {\n        id\n        groupPath\n        ...GroupPackageListItemFragment_package\n        __typename\n      }\n      cursor\n    }\n    pageInfo {\n      endCursor\n      hasNextPage\n      hasPreviousPage\n      startCursor\n    }\n  }\n  id\n}\n\nfragment GroupPackageListItemFragment_package on Package {\n  id\n  name\n  description\n  kind\n  visibility\n  groupPath\n  latestVersion {\n    version\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "6c22bf7b990c76fdd51a97399d4cfe0e";

export default node;
