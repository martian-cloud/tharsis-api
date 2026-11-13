/**
 * @generated SignedSource<<f6490c2df0a158f1c3d603a47d01f808>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type AdminAreaRunnerDetailsQuery$variables = {
  id: string;
};
export type AdminAreaRunnerDetailsQuery$data = {
  readonly node: {
    readonly runner: {
      readonly id: string;
      readonly name: string;
      readonly " $fragmentSpreads": FragmentRefs<"RunnerDetailsFragment_runner">;
    } | null | undefined;
  } | null | undefined;
};
export type AdminAreaRunnerDetailsQuery = {
  response: AdminAreaRunnerDetailsQuery$data;
  variables: AdminAreaRunnerDetailsQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
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
  "name": "id",
  "storageKey": null
},
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "name",
  "storageKey": null
},
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
  "name": "type",
  "storageKey": null
},
v6 = [
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
    "name": "AdminAreaRunnerDetailsQuery",
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
            "fragment": {
              "kind": "InlineFragment",
              "selections": [
                (v2/*:: as any*/),
                (v3/*:: as any*/),
                {
                  "args": null,
                  "kind": "FragmentSpread",
                  "name": "RunnerDetailsFragment_runner"
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
    "argumentDefinitions": (v0/*:: as any*/),
    "kind": "Operation",
    "name": "AdminAreaRunnerDetailsQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*:: as any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          (v4/*:: as any*/),
          (v2/*:: as any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              (v3/*:: as any*/),
              (v5/*:: as any*/),
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
                "name": "description",
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
                  (v4/*:: as any*/),
                  {
                    "kind": "TypeDiscriminator",
                    "abstractKey": "__isActor"
                  },
                  (v5/*:: as any*/),
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
                          (v2/*:: as any*/)
                        ],
                        "storageKey": null
                      }
                    ],
                    "type": "UserActor",
                    "abstractKey": null
                  },
                  {
                    "kind": "InlineFragment",
                    "selections": (v6/*:: as any*/),
                    "type": "ServiceAccountActor",
                    "abstractKey": null
                  },
                  {
                    "kind": "InlineFragment",
                    "selections": (v6/*:: as any*/),
                    "type": "VCSProviderActor",
                    "abstractKey": null
                  },
                  {
                    "kind": "InlineFragment",
                    "selections": (v6/*:: as any*/),
                    "type": "JobActor",
                    "abstractKey": null
                  }
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "tags",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "runUntaggedJobs",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "jobLimit",
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
                    "name": "trn",
                    "storageKey": null
                  }
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": [
                  {
                    "kind": "Literal",
                    "name": "first",
                    "value": 0
                  }
                ],
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
                  }
                ],
                "storageKey": "assignedServiceAccounts(first:0)"
              },
              {
                "alias": null,
                "args": [
                  {
                    "kind": "Literal",
                    "name": "first",
                    "value": 1
                  },
                  {
                    "kind": "Literal",
                    "name": "sort",
                    "value": "LAST_CONTACTED_AT_DESC"
                  }
                ],
                "concreteType": "RunnerSessionConnection",
                "kind": "LinkedField",
                "name": "sessions",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "RunnerSessionEdge",
                    "kind": "LinkedField",
                    "name": "edges",
                    "plural": true,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "RunnerSession",
                        "kind": "LinkedField",
                        "name": "node",
                        "plural": false,
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "active",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "lastContacted",
                            "storageKey": null
                          },
                          (v2/*:: as any*/)
                        ],
                        "storageKey": null
                      }
                    ],
                    "storageKey": null
                  }
                ],
                "storageKey": "sessions(first:1,sort:\"LAST_CONTACTED_AT_DESC\")"
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "resourcePath",
                "storageKey": null
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
    "cacheID": "6135ebaff102fa99bbb55cd59a122d09",
    "id": null,
    "metadata": {},
    "name": "AdminAreaRunnerDetailsQuery",
    "operationKind": "query",
    "text": "query AdminAreaRunnerDetailsQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ... on Runner {\n      id\n      name\n      ...RunnerDetailsFragment_runner\n    }\n    id\n  }\n}\n\nfragment ActorDisplayFragment_actor on Actor {\n  __isActor: __typename\n  type\n  principalId\n  ... on UserActor {\n    user {\n      email\n      id\n    }\n  }\n  ... on ServiceAccountActor {\n    trnSnapshot\n  }\n  ... on VCSProviderActor {\n    trnSnapshot\n  }\n  ... on JobActor {\n    trnSnapshot\n  }\n}\n\nfragment AssignedServiceAccountListFragment_runner on Runner {\n  id\n  resourcePath\n}\n\nfragment RunnerDetailsFragment_runner on Runner {\n  id\n  name\n  type\n  disabled\n  description\n  createdByActor {\n    __typename\n    ...ActorDisplayFragment_actor\n  }\n  tags\n  runUntaggedJobs\n  jobLimit\n  metadata {\n    createdAt\n    trn\n  }\n  assignedServiceAccounts(first: 0) {\n    totalCount\n  }\n  sessions(first: 1, sort: LAST_CONTACTED_AT_DESC) {\n    edges {\n      node {\n        active\n        lastContacted\n        id\n      }\n    }\n  }\n  ...AssignedServiceAccountListFragment_runner\n}\n"
  }
};
})();

(node as any).hash = "287b75c3c28f9c35f606356fd55a2449";

export default node;
