/**
 * @generated SignedSource<<507b5dfd5a60fe8d6e6a1fdb27b4a787>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TerraformModuleVersionAttestListPaginationQuery$variables = {
  after?: string | null | undefined;
  first?: number | null | undefined;
  id: string;
};
export type TerraformModuleVersionAttestListPaginationQuery$data = {
  readonly node: {
    readonly " $fragmentSpreads": FragmentRefs<"TerraformModuleVersionAttestListFragment_attestations">;
  } | null | undefined;
};
export type TerraformModuleVersionAttestListPaginationQuery = {
  response: TerraformModuleVersionAttestListPaginationQuery$data;
  variables: TerraformModuleVersionAttestListPaginationQuery$variables;
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
    "name": "first"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "id"
  }
],
v1 = [
  {
    "kind": "Variable",
    "name": "id",
    "variableName": "id"
  }
],
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "__typename",
  "storageKey": null
},
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v4 = [
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
],
v5 = [
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
    "argumentDefinitions": (v0/*:: as any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "TerraformModuleVersionAttestListPaginationQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*:: as any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          {
            "args": null,
            "kind": "FragmentSpread",
            "name": "TerraformModuleVersionAttestListFragment_attestations"
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
    "argumentDefinitions": (v0/*:: as any*/),
    "kind": "Operation",
    "name": "TerraformModuleVersionAttestListPaginationQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*:: as any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          (v2/*:: as any*/),
          (v3/*:: as any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              {
                "alias": null,
                "args": (v4/*:: as any*/),
                "concreteType": "TerraformModuleAttestationConnection",
                "kind": "LinkedField",
                "name": "attestations",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "TerraformModuleAttestationEdge",
                    "kind": "LinkedField",
                    "name": "edges",
                    "plural": true,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "TerraformModuleAttestation",
                        "kind": "LinkedField",
                        "name": "node",
                        "plural": false,
                        "selections": [
                          (v3/*:: as any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "data",
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
                            "name": "predicateType",
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
                            "concreteType": null,
                            "kind": "LinkedField",
                            "name": "createdByActor",
                            "plural": false,
                            "selections": [
                              (v2/*:: as any*/),
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
                                      (v3/*:: as any*/)
                                    ],
                                    "storageKey": null
                                  }
                                ],
                                "type": "UserActor",
                                "abstractKey": null
                              },
                              {
                                "kind": "InlineFragment",
                                "selections": (v5/*:: as any*/),
                                "type": "ServiceAccountActor",
                                "abstractKey": null
                              },
                              {
                                "kind": "InlineFragment",
                                "selections": (v5/*:: as any*/),
                                "type": "VCSProviderActor",
                                "abstractKey": null
                              },
                              {
                                "kind": "InlineFragment",
                                "selections": (v5/*:: as any*/),
                                "type": "JobActor",
                                "abstractKey": null
                              }
                            ],
                            "storageKey": null
                          },
                          (v2/*:: as any*/)
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
                "args": (v4/*:: as any*/),
                "filters": null,
                "handle": "connection",
                "key": "TerraformModuleVersionAttestList_attestations",
                "kind": "LinkedHandle",
                "name": "attestations"
              }
            ],
            "type": "TerraformModuleVersion",
            "abstractKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "89868ff7322903a0d84eda114ab5dda8",
    "id": null,
    "metadata": {},
    "name": "TerraformModuleVersionAttestListPaginationQuery",
    "operationKind": "query",
    "text": "query TerraformModuleVersionAttestListPaginationQuery(\n  $after: String\n  $first: Int\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ...TerraformModuleVersionAttestListFragment_attestations\n    id\n  }\n}\n\nfragment ActorDisplayFragment_actor on Actor {\n  __isActor: __typename\n  type\n  principalId\n  ... on UserActor {\n    user {\n      email\n      id\n    }\n  }\n  ... on ServiceAccountActor {\n    trnSnapshot\n  }\n  ... on VCSProviderActor {\n    trnSnapshot\n  }\n  ... on JobActor {\n    trnSnapshot\n  }\n}\n\nfragment TerraformModuleVersionAttestListFragment_attestations on TerraformModuleVersion {\n  id\n  attestations(first: $first, after: $after) {\n    edges {\n      node {\n        id\n        data\n        ...TerraformModuleVersionAttestListItemFragment_module\n        __typename\n      }\n      cursor\n    }\n    pageInfo {\n      endCursor\n      hasNextPage\n    }\n  }\n}\n\nfragment TerraformModuleVersionAttestListItemFragment_module on TerraformModuleAttestation {\n  id\n  description\n  predicateType\n  data\n  metadata {\n    createdAt\n  }\n  createdByActor {\n    __typename\n    ...ActorDisplayFragment_actor\n  }\n}\n"
  }
};
})();

(node as any).hash = "0642f53265933eb3763b69621c892c14";

export default node;
