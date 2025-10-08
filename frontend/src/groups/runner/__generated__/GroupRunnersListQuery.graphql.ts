/**
 * @generated SignedSource<<3736155f4530eea4fca147c73d69e9ea>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupRunnersListQuery$variables = {
  after?: string | null | undefined;
  first: number;
  id: string;
};
export type GroupRunnersListQuery$data = {
  readonly node: {
    readonly fullPath?: string;
    readonly " $fragmentSpreads": FragmentRefs<"GroupRunnersListFragment_runners">;
  } | null | undefined;
};
export type GroupRunnersListQuery = {
  response: GroupRunnersListQuery$data;
  variables: GroupRunnersListQuery$variables;
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
  "name": "fullPath",
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
  "name": "id",
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
  },
  {
    "kind": "Literal",
    "name": "includeInherited",
    "value": true
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
      (v2/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "GroupRunnersListQuery",
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
                "name": "GroupRunnersListFragment_runners"
              },
              (v4/*: any*/)
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
      (v1/*: any*/),
      (v0/*: any*/)
    ],
    "kind": "Operation",
    "name": "GroupRunnersListQuery",
    "selections": [
      {
        "alias": null,
        "args": (v3/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          (v5/*: any*/),
          (v6/*: any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              {
                "alias": null,
                "args": (v7/*: any*/),
                "concreteType": "RunnerConnection",
                "kind": "LinkedField",
                "name": "runners",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "RunnerEdge",
                    "kind": "LinkedField",
                    "name": "edges",
                    "plural": true,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "Runner",
                        "kind": "LinkedField",
                        "name": "node",
                        "plural": false,
                        "selections": [
                          (v6/*: any*/),
                          (v5/*: any*/),
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
                                "name": "createdAt",
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
                            "name": "disabled",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "createdBy",
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
                  "includeInherited",
                  "sort"
                ],
                "handle": "connection",
                "key": "GroupRunnersList_runners",
                "kind": "LinkedHandle",
                "name": "runners"
              },
              (v4/*: any*/)
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
    "cacheID": "d011ee035a734a893c503dd007acc112",
    "id": null,
    "metadata": {},
    "name": "GroupRunnersListQuery",
    "operationKind": "query",
    "text": "query GroupRunnersListQuery(\n  $id: String!\n  $first: Int!\n  $after: String\n) {\n  node(id: $id) {\n    __typename\n    ... on Group {\n      ...GroupRunnersListFragment_runners\n      fullPath\n    }\n    id\n  }\n}\n\nfragment GroupRunnersListFragment_runners on Group {\n  runners(after: $after, first: $first, includeInherited: true, sort: GROUP_LEVEL_DESC) {\n    edges {\n      node {\n        id\n        __typename\n      }\n      cursor\n    }\n    ...RunnerListFragment_runners\n    pageInfo {\n      endCursor\n      hasNextPage\n    }\n  }\n  id\n}\n\nfragment RunnerListFragment_runners on RunnerConnection {\n  edges {\n    node {\n      id\n      groupPath\n      ...RunnerListItemFragment_runner\n    }\n  }\n}\n\nfragment RunnerListItemFragment_runner on Runner {\n  metadata {\n    createdAt\n  }\n  id\n  name\n  disabled\n  createdBy\n  groupPath\n}\n"
  }
};
})();

(node as any).hash = "f51bb3b4a30e8eecc48a42d495f74549";

export default node;
