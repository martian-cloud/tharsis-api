/**
 * @generated SignedSource<<7d5803183c81995e24b837dcd6a667c4>>
 * @lightSyntaxTransform
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
    readonly group: {
      readonly fullPath: string;
      readonly " $fragmentSpreads": FragmentRefs<"GroupRunnersListFragment_runners">;
    } | null | undefined;
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
],
v8 = [
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "trnSnapshot",
    "storageKey": null
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
    "name": "GroupRunnersListQuery",
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
                  "name": "GroupRunnersListFragment_runners"
                },
                (v4/*:: as any*/)
              ],
              "type": "Group",
              "abstractKey": null
            },
            "kind": "AliasedInlineFragmentSpread",
            "name": "group"
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
    "name": "GroupRunnersListQuery",
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
          (v6/*:: as any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              {
                "alias": null,
                "args": (v7/*:: as any*/),
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
                          (v6/*:: as any*/),
                          (v5/*:: as any*/),
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
                              },
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
                            "name": "disabled",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": null,
                            "kind": "LinkedField",
                            "name": "createdByActor",
                            "plural": false,
                            "selections": [
                              (v5/*:: as any*/),
                              {
                                "kind": "TypeDiscriminator",
                                "abstractKey": "__isActor"
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
                                "name": "principalId",
                                "storageKey": null
                              },
                              {
                                "kind": "InlineFragment",
                                "selections": [
                                  {
                                    "alias": null,
                                    "args": null,
                                    "concreteType": "User",
                                    "kind": "LinkedField",
                                    "name": "user",
                                    "plural": false,
                                    "selections": [
                                      {
                                        "alias": null,
                                        "args": null,
                                        "kind": "ScalarField",
                                        "name": "email",
                                        "storageKey": null
                                      },
                                      (v6/*:: as any*/)
                                    ],
                                    "storageKey": null
                                  }
                                ],
                                "type": "UserActor",
                                "abstractKey": null
                              },
                              {
                                "kind": "InlineFragment",
                                "selections": (v8/*:: as any*/),
                                "type": "ServiceAccountActor",
                                "abstractKey": null
                              },
                              {
                                "kind": "InlineFragment",
                                "selections": (v8/*:: as any*/),
                                "type": "VCSProviderActor",
                                "abstractKey": null
                              },
                              {
                                "kind": "InlineFragment",
                                "selections": (v8/*:: as any*/),
                                "type": "JobActor",
                                "abstractKey": null
                              }
                            ],
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
                "args": (v7/*:: as any*/),
                "filters": [
                  "includeInherited",
                  "sort"
                ],
                "handle": "connection",
                "key": "GroupRunnersList_runners",
                "kind": "LinkedHandle",
                "name": "runners"
              },
              (v4/*:: as any*/)
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
    "cacheID": "33d237a8a545cd26149f16b4e38b5fd4",
    "id": null,
    "metadata": {},
    "name": "GroupRunnersListQuery",
    "operationKind": "query",
    "text": "query GroupRunnersListQuery(\n  $id: String!\n  $first: Int!\n  $after: String\n) {\n  node(id: $id) {\n    __typename\n    ... on Group {\n      ...GroupRunnersListFragment_runners\n      fullPath\n    }\n    id\n  }\n}\n\nfragment ActorDisplayFragment_actor on Actor {\n  __isActor: __typename\n  type\n  principalId\n  ... on UserActor {\n    user {\n      email\n      id\n    }\n  }\n  ... on ServiceAccountActor {\n    trnSnapshot\n  }\n  ... on VCSProviderActor {\n    trnSnapshot\n  }\n  ... on JobActor {\n    trnSnapshot\n  }\n}\n\nfragment GroupRunnersListFragment_runners on Group {\n  runners(after: $after, first: $first, includeInherited: true, sort: GROUP_LEVEL_DESC) {\n    edges {\n      node {\n        id\n        __typename\n      }\n      cursor\n    }\n    ...RunnerListFragment_runners\n    pageInfo {\n      endCursor\n      hasNextPage\n    }\n  }\n  id\n}\n\nfragment RunnerListFragment_runners on RunnerConnection {\n  edges {\n    node {\n      id\n      groupPath\n      ...RunnerListItemFragment_runner\n    }\n  }\n}\n\nfragment RunnerListItemFragment_runner on Runner {\n  metadata {\n    createdAt\n    updatedAt\n  }\n  id\n  name\n  disabled\n  createdByActor {\n    __typename\n    ...ActorDisplayFragment_actor\n  }\n  groupPath\n}\n"
  }
};
})();

(node as any).hash = "0f93160e94311f117bd3f2acc7f926d5";

export default node;
