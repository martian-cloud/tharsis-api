/**
 * @generated SignedSource<<253188ff0f37b47834cd1536219784cc>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ManagedIdentityListQuery$variables = {
  after?: string | null | undefined;
  before?: string | null | undefined;
  first?: number | null | undefined;
  groupId: string;
  last?: number | null | undefined;
  search?: string | null | undefined;
};
export type ManagedIdentityListQuery$data = {
  readonly node: {
    readonly " $fragmentSpreads": FragmentRefs<"ManagedIdentityListFragment_managedIdentities">;
  } | null | undefined;
};
export type ManagedIdentityListQuery = {
  response: ManagedIdentityListQuery$data;
  variables: ManagedIdentityListQuery$variables;
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
v9 = [
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
    "kind": "Literal",
    "name": "includeInherited",
    "value": true
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
    "name": "ManagedIdentityListQuery",
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
                "name": "ManagedIdentityListFragment_managedIdentities"
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
    "name": "ManagedIdentityListQuery",
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
                "alias": null,
                "args": (v9/*: any*/),
                "concreteType": "ManagedIdentityConnection",
                "kind": "LinkedField",
                "name": "managedIdentities",
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
                    "concreteType": "ManagedIdentityEdge",
                    "kind": "LinkedField",
                    "name": "edges",
                    "plural": true,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "ManagedIdentity",
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
                            "name": "isAlias",
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
                            "name": "type",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "resourcePath",
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
                  }
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": (v9/*: any*/),
                "filters": [
                  "search",
                  "includeInherited",
                  "sort"
                ],
                "handle": "connection",
                "key": "ManagedIdentityList_managedIdentities",
                "kind": "LinkedHandle",
                "name": "managedIdentities"
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
    "cacheID": "07ac1db2b4bdbb03a6fa296a2d6b4967",
    "id": null,
    "metadata": {},
    "name": "ManagedIdentityListQuery",
    "operationKind": "query",
    "text": "query ManagedIdentityListQuery(\n  $first: Int\n  $last: Int\n  $after: String\n  $before: String\n  $groupId: String!\n  $search: String\n) {\n  node(id: $groupId) {\n    __typename\n    ... on Group {\n      ...ManagedIdentityListFragment_managedIdentities\n    }\n    id\n  }\n}\n\nfragment ManagedIdentityListFragment_managedIdentities on Group {\n  managedIdentities(after: $after, before: $before, first: $first, last: $last, search: $search, includeInherited: true, sort: GROUP_LEVEL_DESC) {\n    totalCount\n    edges {\n      node {\n        id\n        groupPath\n        ...ManagedIdentityListItemFragment_managedIdentity\n        __typename\n      }\n      cursor\n    }\n    pageInfo {\n      endCursor\n      hasNextPage\n      hasPreviousPage\n      startCursor\n    }\n  }\n  id\n}\n\nfragment ManagedIdentityListItemFragment_managedIdentity on ManagedIdentity {\n  metadata {\n    updatedAt\n  }\n  id\n  isAlias\n  name\n  description\n  type\n  resourcePath\n  groupPath\n}\n"
  }
};
})();

(node as any).hash = "cd741d437a029d760c0b5601bd23f04a";

export default node;
