/**
 * @generated SignedSource<<456d6ee5b85fd367dce7389bfbc23455>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupOrWorkspaceRendererQuery$variables = {
  fullPath: string;
};
export type GroupOrWorkspaceRendererQuery$data = {
  readonly namespace: {
    readonly __typename: string;
    readonly fullPath: string;
    readonly id: string;
    readonly " $fragmentSpreads": FragmentRefs<"GroupDetailsFragment_group" | "WorkspaceDetailsFragment_workspace">;
  } | null | undefined;
};
export type GroupOrWorkspaceRendererQuery = {
  response: GroupOrWorkspaceRendererQuery$data;
  variables: GroupOrWorkspaceRendererQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "fullPath"
  }
],
v1 = [
  {
    "kind": "Variable",
    "name": "fullPath",
    "variableName": "fullPath"
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
  "name": "name",
  "storageKey": null
},
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "description",
  "storageKey": null
},
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "trn",
  "storageKey": null
},
v8 = {
  "alias": null,
  "args": null,
  "concreteType": "ResourceMetadata",
  "kind": "LinkedField",
  "name": "metadata",
  "plural": false,
  "selections": [
    (v7/*: any*/)
  ],
  "storageKey": null
},
v9 = {
  "kind": "Literal",
  "name": "first",
  "value": 0
},
v10 = [
  (v9/*: any*/)
],
v11 = [
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "totalCount",
    "storageKey": null
  }
],
v12 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "inherited",
  "storageKey": null
},
v13 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "namespacePath",
  "storageKey": null
},
v14 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "value",
  "storageKey": null
},
v15 = {
  "alias": null,
  "args": null,
  "concreteType": "NamespaceRunnerTags",
  "kind": "LinkedField",
  "name": "runnerTags",
  "plural": false,
  "selections": [
    (v12/*: any*/),
    (v13/*: any*/),
    (v14/*: any*/)
  ],
  "storageKey": null
},
v16 = {
  "alias": null,
  "args": null,
  "concreteType": "NamespaceDriftDetectionEnabled",
  "kind": "LinkedField",
  "name": "driftDetectionEnabled",
  "plural": false,
  "selections": [
    (v12/*: any*/),
    (v14/*: any*/),
    (v13/*: any*/)
  ],
  "storageKey": null
},
v17 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "key",
  "storageKey": null
},
v18 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "category",
  "storageKey": null
},
v19 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "sensitive",
  "storageKey": null
},
v20 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "updatedAt",
  "storageKey": null
},
v21 = {
  "alias": null,
  "args": null,
  "concreteType": "ResourceMetadata",
  "kind": "LinkedField",
  "name": "metadata",
  "plural": false,
  "selections": [
    (v20/*: any*/)
  ],
  "storageKey": null
},
v22 = [
  (v5/*: any*/),
  (v3/*: any*/)
],
v23 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "resourcePath",
  "storageKey": null
},
v24 = [
  (v3/*: any*/)
],
v25 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "createdAt",
  "storageKey": null
},
v26 = {
  "kind": "InlineFragment",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "NamespaceVariable",
      "kind": "LinkedField",
      "name": "variables",
      "plural": true,
      "selections": [
        (v3/*: any*/),
        (v17/*: any*/),
        (v18/*: any*/),
        (v19/*: any*/),
        (v14/*: any*/),
        (v13/*: any*/),
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "latestVersionId",
          "storageKey": null
        },
        (v21/*: any*/)
      ],
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "NamespaceMembership",
      "kind": "LinkedField",
      "name": "memberships",
      "plural": true,
      "selections": [
        (v3/*: any*/),
        {
          "alias": null,
          "args": null,
          "concreteType": null,
          "kind": "LinkedField",
          "name": "member",
          "plural": false,
          "selections": [
            (v2/*: any*/),
            {
              "kind": "InlineFragment",
              "selections": [
                {
                  "alias": null,
                  "args": null,
                  "kind": "ScalarField",
                  "name": "username",
                  "storageKey": null
                },
                {
                  "alias": null,
                  "args": null,
                  "kind": "ScalarField",
                  "name": "email",
                  "storageKey": null
                },
                (v3/*: any*/)
              ],
              "type": "User",
              "abstractKey": null
            },
            {
              "kind": "InlineFragment",
              "selections": (v22/*: any*/),
              "type": "Team",
              "abstractKey": null
            },
            {
              "kind": "InlineFragment",
              "selections": [
                (v23/*: any*/),
                (v5/*: any*/),
                (v3/*: any*/)
              ],
              "type": "ServiceAccount",
              "abstractKey": null
            },
            {
              "kind": "InlineFragment",
              "selections": (v24/*: any*/),
              "type": "Node",
              "abstractKey": "__isNode"
            }
          ],
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
            (v25/*: any*/),
            (v20/*: any*/),
            (v7/*: any*/)
          ],
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "concreteType": "Role",
          "kind": "LinkedField",
          "name": "role",
          "plural": false,
          "selections": (v22/*: any*/),
          "storageKey": null
        },
        (v23/*: any*/)
      ],
      "storageKey": null
    }
  ],
  "type": "Namespace",
  "abstractKey": "__isNamespace"
},
v27 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "type",
  "storageKey": null
},
v28 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "status",
  "storageKey": null
},
v29 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "createdBy",
  "storageKey": null
},
v30 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "isDestroy",
  "storageKey": null
},
v31 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "moduleSource",
  "storageKey": null
},
v32 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "moduleVersion",
  "storageKey": null
},
v33 = {
  "alias": null,
  "args": null,
  "concreteType": "ResourceMetadata",
  "kind": "LinkedField",
  "name": "metadata",
  "plural": false,
  "selections": [
    (v25/*: any*/)
  ],
  "storageKey": null
},
v34 = {
  "alias": null,
  "args": null,
  "concreteType": "Plan",
  "kind": "LinkedField",
  "name": "plan",
  "plural": false,
  "selections": [
    (v28/*: any*/),
    (v33/*: any*/),
    (v3/*: any*/)
  ],
  "storageKey": null
},
v35 = {
  "alias": null,
  "args": null,
  "concreteType": "Apply",
  "kind": "LinkedField",
  "name": "apply",
  "plural": false,
  "selections": [
    (v28/*: any*/),
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "triggeredBy",
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
        (v25/*: any*/),
        (v20/*: any*/)
      ],
      "storageKey": null
    },
    (v3/*: any*/)
  ],
  "storageKey": null
},
v36 = {
  "kind": "Literal",
  "name": "includeInherited",
  "value": true
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "GroupOrWorkspaceRendererQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "namespace",
        "plural": false,
        "selections": [
          (v2/*: any*/),
          (v3/*: any*/),
          (v4/*: any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "GroupDetailsFragment_group"
              }
            ],
            "type": "Group",
            "abstractKey": null
          },
          {
            "kind": "InlineFragment",
            "selections": [
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "WorkspaceDetailsFragment_workspace"
              }
            ],
            "type": "Workspace",
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
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "GroupOrWorkspaceRendererQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "namespace",
        "plural": false,
        "selections": [
          (v2/*: any*/),
          (v3/*: any*/),
          (v4/*: any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              (v5/*: any*/),
              (v6/*: any*/),
              (v8/*: any*/),
              {
                "alias": null,
                "args": (v10/*: any*/),
                "concreteType": "WorkspaceConnection",
                "kind": "LinkedField",
                "name": "workspaces",
                "plural": false,
                "selections": (v11/*: any*/),
                "storageKey": "workspaces(first:0)"
              },
              {
                "alias": null,
                "args": (v10/*: any*/),
                "concreteType": "GroupConnection",
                "kind": "LinkedField",
                "name": "descendentGroups",
                "plural": false,
                "selections": (v11/*: any*/),
                "storageKey": "descendentGroups(first:0)"
              },
              (v15/*: any*/),
              (v16/*: any*/),
              (v26/*: any*/)
            ],
            "type": "Group",
            "abstractKey": null
          },
          {
            "kind": "InlineFragment",
            "selections": [
              (v5/*: any*/),
              (v6/*: any*/),
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "preventDestroyPlan",
                "storageKey": null
              },
              (v8/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "WorkspaceAssessment",
                "kind": "LinkedField",
                "name": "assessment",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "hasDrift",
                    "storageKey": null
                  },
                  (v3/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "startedAt",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "completedAt",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "Run",
                    "kind": "LinkedField",
                    "name": "run",
                    "plural": false,
                    "selections": (v24/*: any*/),
                    "storageKey": null
                  }
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "Job",
                "kind": "LinkedField",
                "name": "currentJob",
                "plural": false,
                "selections": [
                  (v3/*: any*/),
                  (v27/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "Run",
                    "kind": "LinkedField",
                    "name": "run",
                    "plural": false,
                    "selections": [
                      (v3/*: any*/),
                      (v28/*: any*/),
                      (v29/*: any*/),
                      (v30/*: any*/),
                      (v31/*: any*/),
                      (v32/*: any*/),
                      (v33/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "ConfigurationVersion",
                        "kind": "LinkedField",
                        "name": "configurationVersion",
                        "plural": false,
                        "selections": (v24/*: any*/),
                        "storageKey": null
                      },
                      (v34/*: any*/),
                      (v35/*: any*/)
                    ],
                    "storageKey": null
                  }
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "StateVersion",
                "kind": "LinkedField",
                "name": "currentStateVersion",
                "plural": false,
                "selections": [
                  (v3/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "StateVersionOutput",
                    "kind": "LinkedField",
                    "name": "outputs",
                    "plural": true,
                    "selections": [
                      (v5/*: any*/),
                      (v14/*: any*/),
                      (v27/*: any*/),
                      (v19/*: any*/),
                      (v3/*: any*/)
                    ],
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "StateVersionResource",
                    "kind": "LinkedField",
                    "name": "resources",
                    "plural": true,
                    "selections": [
                      (v5/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "provider",
                        "storageKey": null
                      },
                      (v27/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "mode",
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "module",
                        "storageKey": null
                      }
                    ],
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "StateVersionDependency",
                    "kind": "LinkedField",
                    "name": "dependencies",
                    "plural": true,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "workspacePath",
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "StateVersion",
                        "kind": "LinkedField",
                        "name": "stateVersion",
                        "plural": false,
                        "selections": [
                          (v3/*: any*/),
                          (v21/*: any*/)
                        ],
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "Workspace",
                        "kind": "LinkedField",
                        "name": "workspace",
                        "plural": false,
                        "selections": [
                          (v3/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": "StateVersion",
                            "kind": "LinkedField",
                            "name": "currentStateVersion",
                            "plural": false,
                            "selections": (v24/*: any*/),
                            "storageKey": null
                          }
                        ],
                        "storageKey": null
                      }
                    ],
                    "storageKey": null
                  },
                  (v33/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "Run",
                    "kind": "LinkedField",
                    "name": "run",
                    "plural": false,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "RunVariable",
                        "kind": "LinkedField",
                        "name": "variables",
                        "plural": true,
                        "selections": [
                          (v17/*: any*/),
                          (v18/*: any*/),
                          (v13/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "includedInTfConfig",
                            "storageKey": null
                          },
                          (v14/*: any*/),
                          (v19/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "versionId",
                            "storageKey": null
                          }
                        ],
                        "storageKey": null
                      },
                      (v3/*: any*/),
                      (v28/*: any*/),
                      (v29/*: any*/),
                      (v30/*: any*/),
                      (v31/*: any*/),
                      (v32/*: any*/),
                      (v33/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "ConfigurationVersion",
                        "kind": "LinkedField",
                        "name": "configurationVersion",
                        "plural": false,
                        "selections": [
                          (v3/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": "VCSEvent",
                            "kind": "LinkedField",
                            "name": "vcsEvent",
                            "plural": false,
                            "selections": [
                              (v28/*: any*/),
                              (v3/*: any*/)
                            ],
                            "storageKey": null
                          }
                        ],
                        "storageKey": null
                      },
                      (v34/*: any*/),
                      (v35/*: any*/)
                    ],
                    "storageKey": null
                  }
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": [
                  (v9/*: any*/),
                  (v36/*: any*/)
                ],
                "concreteType": "ManagedIdentityConnection",
                "kind": "LinkedField",
                "name": "managedIdentities",
                "plural": false,
                "selections": (v11/*: any*/),
                "storageKey": "managedIdentities(first:0,includeInherited:true)"
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "ManagedIdentity",
                "kind": "LinkedField",
                "name": "assignedManagedIdentities",
                "plural": true,
                "selections": [
                  (v3/*: any*/),
                  (v21/*: any*/),
                  (v5/*: any*/),
                  (v6/*: any*/),
                  (v27/*: any*/),
                  (v23/*: any*/)
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "WorkspaceVCSProviderLink",
                "kind": "LinkedField",
                "name": "workspaceVcsProviderLink",
                "plural": false,
                "selections": [
                  (v3/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "branch",
                    "storageKey": null
                  },
                  (v33/*: any*/),
                  (v29/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "repositoryPath",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "autoSpeculativePlan",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "webhookDisabled",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "moduleDirectory",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "tagRegex",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "globPatterns",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "VCSProvider",
                    "kind": "LinkedField",
                    "name": "vcsProvider",
                    "plural": false,
                    "selections": [
                      (v3/*: any*/),
                      (v5/*: any*/),
                      (v6/*: any*/),
                      (v27/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "autoCreateWebhooks",
                        "storageKey": null
                      }
                    ],
                    "storageKey": null
                  }
                ],
                "storageKey": null
              },
              (v15/*: any*/),
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "maxJobDuration",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "terraformVersion",
                "storageKey": null
              },
              (v16/*: any*/),
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "groupPath",
                "storageKey": null
              },
              {
                "alias": null,
                "args": [
                  {
                    "kind": "Literal",
                    "name": "first",
                    "value": 10
                  },
                  (v36/*: any*/)
                ],
                "concreteType": "VCSProviderConnection",
                "kind": "LinkedField",
                "name": "vcsProviders",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "VCSProviderEdge",
                    "kind": "LinkedField",
                    "name": "edges",
                    "plural": true,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "VCSProvider",
                        "kind": "LinkedField",
                        "name": "node",
                        "plural": false,
                        "selections": (v24/*: any*/),
                        "storageKey": null
                      }
                    ],
                    "storageKey": null
                  }
                ],
                "storageKey": "vcsProviders(first:10,includeInherited:true)"
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "locked",
                "storageKey": null
              },
              (v26/*: any*/)
            ],
            "type": "Workspace",
            "abstractKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "efed72b7a00d156aa809bde48eaab768",
    "id": null,
    "metadata": {},
    "name": "GroupOrWorkspaceRendererQuery",
    "operationKind": "query",
    "text": "query GroupOrWorkspaceRendererQuery(\n  $fullPath: String!\n) {\n  namespace(fullPath: $fullPath) {\n    __typename\n    id\n    fullPath\n    ... on Group {\n      ...GroupDetailsFragment_group\n    }\n    ... on Workspace {\n      ...WorkspaceDetailsFragment_workspace\n    }\n  }\n}\n\nfragment AssignedManagedIdentityListFragment_assignedManagedIdentities on Workspace {\n  id\n  fullPath\n  managedIdentities(includeInherited: true, first: 0) {\n    totalCount\n  }\n  assignedManagedIdentities {\n    id\n    ...AssignedManagedIdentityListItemFragment_managedIdentity\n  }\n}\n\nfragment AssignedManagedIdentityListItemFragment_managedIdentity on ManagedIdentity {\n  metadata {\n    updatedAt\n  }\n  id\n  name\n  description\n  type\n  resourcePath\n}\n\nfragment CreateRunFragment_workspace on Workspace {\n  id\n  fullPath\n  workspaceVcsProviderLink {\n    id\n  }\n  ...ModuleSourceFragment_workspace\n  ...VCSWorkspaceLinkSourceFragment_workspace\n}\n\nfragment DriftDetectionSettingsFormFragment_driftDetectionEnabled on NamespaceDriftDetectionEnabled {\n  inherited\n  namespacePath\n  value\n}\n\nfragment EditFederatedRegistryFragment_group on Group {\n  fullPath\n}\n\nfragment EditGroupRunnerFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment EditManagedIdentityFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment EditServiceAccountFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment EditVCSProviderFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment EditVCSProviderLinkFragment_workspace on Workspace {\n  fullPath\n  workspaceVcsProviderLink {\n    id\n    metadata {\n      createdAt\n    }\n    createdBy\n    repositoryPath\n    autoSpeculativePlan\n    webhookDisabled\n    moduleDirectory\n    branch\n    tagRegex\n    globPatterns\n    vcsProvider {\n      id\n      name\n      description\n      type\n      autoCreateWebhooks\n    }\n  }\n  ...VCSProviderLinkFormFragment_workspace\n}\n\nfragment EditVCSProviderOAuthCredentialsFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment FederatedRegistriesFragment_group on Group {\n  ...FederatedRegistryListFragment_group\n  ...FederatedRegistryDetailsFragment_group\n  ...NewFederatedRegistryFragment_group\n  ...EditFederatedRegistryFragment_group\n}\n\nfragment FederatedRegistryDetailsFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment FederatedRegistryListFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment GPGKeyListFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment GPGKeysFragment_group on Group {\n  ...GPGKeyListFragment_group\n  ...NewGPGKeyFragment_group\n}\n\nfragment GroupAdvancedSettingsDeleteDialogFragment_group on Group {\n  name\n  fullPath\n}\n\nfragment GroupAdvancedSettingsFragment_group on Group {\n  name\n  fullPath\n  ...GroupAdvancedSettingsDeleteDialogFragment_group\n  ...MigrateGroupDialogFragment_group\n}\n\nfragment GroupDetailsFragment_group on Group {\n  id\n  fullPath\n  name\n  ...GroupDetailsIndexFragment_group\n  ...ManagedIdentitiesFragment_group\n  ...GroupRunnersFragment_group\n  ...ServiceAccountsFragment_group\n  ...VCSProvidersFragment_group\n  ...FederatedRegistriesFragment_group\n  ...VariablesFragment_variables\n  ...NamespaceMembershipsFragment_memberships\n  ...GPGKeysFragment_group\n  ...NamespaceActivityFragment_activity\n  ...GroupSettingsFragment_group\n}\n\nfragment GroupDetailsIndexFragment_group on Group {\n  id\n  name\n  description\n  fullPath\n  metadata {\n    trn\n  }\n  workspaces(first: 0) {\n    totalCount\n  }\n  descendentGroups(first: 0) {\n    totalCount\n  }\n  ...WorkspaceListFragment_group\n  ...MigrateGroupDialogFragment_group\n  ...GroupNotificationPreferenceFragment_group\n}\n\nfragment GroupDriftDetectionSettingsFragment_group on Group {\n  fullPath\n  driftDetectionEnabled {\n    inherited\n    value\n    ...DriftDetectionSettingsFormFragment_driftDetectionEnabled\n  }\n}\n\nfragment GroupGeneralSettingsFragment_group on Group {\n  name\n  description\n  fullPath\n}\n\nfragment GroupNotificationPreferenceFragment_group on Group {\n  fullPath\n}\n\nfragment GroupRunnerDetailsFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment GroupRunnerSettingsFragment_group on Group {\n  fullPath\n  runnerTags {\n    inherited\n    namespacePath\n    value\n    ...RunnerSettingsForm_runnerTags\n  }\n}\n\nfragment GroupRunnersFragment_group on Group {\n  ...GroupRunnersListFragment_group\n  ...NewGroupRunnerFragment_group\n  ...EditGroupRunnerFragment_group\n  ...GroupRunnerDetailsFragment_group\n}\n\nfragment GroupRunnersListFragment_group on Group {\n  id\n}\n\nfragment GroupSettingsFragment_group on Group {\n  fullPath\n  ...GroupGeneralSettingsFragment_group\n  ...GroupAdvancedSettingsFragment_group\n  ...GroupRunnerSettingsFragment_group\n  ...GroupDriftDetectionSettingsFragment_group\n}\n\nfragment ManagedIdentitiesFragment_group on Group {\n  ...ManagedIdentityListFragment_group\n  ...NewManagedIdentityFragment_group\n  ...EditManagedIdentityFragment_group\n  ...ManagedIdentityDetailsFragment_group\n}\n\nfragment ManagedIdentityDetailsFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment ManagedIdentityListFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment MaxJobDurationSettingFragment_workspace on Workspace {\n  maxJobDuration\n}\n\nfragment MigrateGroupDialogFragment_group on Group {\n  name\n  fullPath\n}\n\nfragment MigrateWorkspaceDialogFragment_workspace on Workspace {\n  name\n  fullPath\n  groupPath\n}\n\nfragment ModuleSourceFragment_workspace on Workspace {\n  fullPath\n}\n\nfragment NamespaceActivityFragment_activity on Namespace {\n  __isNamespace: __typename\n  __typename\n  fullPath\n}\n\nfragment NamespaceMembershipListFragment_memberships on Namespace {\n  __isNamespace: __typename\n  fullPath\n  memberships {\n    id\n    member {\n      __typename\n      ... on User {\n        username\n        email\n      }\n      ... on Team {\n        name\n      }\n      ... on ServiceAccount {\n        resourcePath\n        name\n      }\n      ... on Node {\n        __isNode: __typename\n        id\n      }\n    }\n    ...NamespaceMembershipListItemFragment_membership\n  }\n}\n\nfragment NamespaceMembershipListItemFragment_membership on NamespaceMembership {\n  metadata {\n    createdAt\n    updatedAt\n    trn\n  }\n  id\n  role {\n    name\n    id\n  }\n  resourcePath\n  member {\n    __typename\n    ... on User {\n      id\n      username\n      email\n    }\n    ... on Team {\n      id\n      name\n    }\n    ... on ServiceAccount {\n      id\n      name\n      resourcePath\n    }\n    ... on Node {\n      __isNode: __typename\n      id\n    }\n  }\n}\n\nfragment NamespaceMembershipsFragment_memberships on Namespace {\n  __isNamespace: __typename\n  ...NamespaceMembershipsIndexFragment_memberships\n  ...NewNamespaceMembershipFragment_memberships\n}\n\nfragment NamespaceMembershipsIndexFragment_memberships on Namespace {\n  __isNamespace: __typename\n  fullPath\n  ...NamespaceMembershipListFragment_memberships\n}\n\nfragment NewFederatedRegistryFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment NewGPGKeyFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment NewGroupRunnerFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment NewManagedIdentityFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment NewNamespaceMembershipFragment_memberships on Namespace {\n  __isNamespace: __typename\n  fullPath\n}\n\nfragment NewServiceAccountFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment NewVCSProviderFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment NewVCSProviderLinkFragment_workspace on Workspace {\n  fullPath\n  ...VCSProviderLinkFormFragment_workspace\n}\n\nfragment RunDetailsFragment_details on Workspace {\n  id\n  fullPath\n}\n\nfragment RunnerSettingsForm_runnerTags on NamespaceRunnerTags {\n  inherited\n  namespacePath\n  value\n}\n\nfragment RunsFragment_runs on Workspace {\n  fullPath\n  ...RunsIndexFragment_runs\n  ...CreateRunFragment_workspace\n  ...RunDetailsFragment_details\n}\n\nfragment RunsIndexFragment_runs on Workspace {\n  id\n  fullPath\n}\n\nfragment ServiceAccountDetailsFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment ServiceAccountListFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment ServiceAccountsFragment_group on Group {\n  ...ServiceAccountListFragment_group\n  ...ServiceAccountDetailsFragment_group\n  ...NewServiceAccountFragment_group\n  ...EditServiceAccountFragment_group\n}\n\nfragment StateVersionDependenciesFragment_dependencies on StateVersion {\n  dependencies {\n    workspacePath\n    ...StateVersionDependencyListItemFragment_dependency\n  }\n}\n\nfragment StateVersionDependencyListItemFragment_dependency on StateVersionDependency {\n  workspacePath\n  stateVersion {\n    id\n    metadata {\n      updatedAt\n    }\n  }\n  workspace {\n    id\n    currentStateVersion {\n      id\n    }\n  }\n}\n\nfragment StateVersionDetailsFragment_details on Workspace {\n  id\n  fullPath\n}\n\nfragment StateVersionFileFragment_stateVersion on StateVersion {\n  id\n}\n\nfragment StateVersionInputVariableListItemFragment_variable on RunVariable {\n  key\n  value\n  category\n  namespacePath\n  sensitive\n  versionId\n  includedInTfConfig\n}\n\nfragment StateVersionInputVariablesFragment_variables on Run {\n  variables {\n    key\n    category\n    namespacePath\n    includedInTfConfig\n    ...StateVersionInputVariableListItemFragment_variable\n  }\n}\n\nfragment StateVersionListFragment_workspace on Workspace {\n  id\n  fullPath\n}\n\nfragment StateVersionOutputListItemFragment_output on StateVersionOutput {\n  name\n  value\n  type\n  sensitive\n}\n\nfragment StateVersionOutputsFragment_outputs on StateVersion {\n  outputs {\n    name\n    ...StateVersionOutputListItemFragment_output\n    id\n  }\n}\n\nfragment StateVersionResourceListItemFragment_resource on StateVersionResource {\n  name\n  type\n  provider\n  mode\n  module\n}\n\nfragment StateVersionResourcesFragment_resources on StateVersion {\n  resources {\n    name\n    provider\n    type\n    ...StateVersionResourceListItemFragment_resource\n  }\n}\n\nfragment StateVersionsFragment_stateVersions on Workspace {\n  fullPath\n  ...StateVersionListFragment_workspace\n  ...StateVersionDetailsFragment_details\n}\n\nfragment VCSProviderDetailsFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment VCSProviderLinkFormFragment_workspace on Workspace {\n  fullPath\n  workspaceVcsProviderLink {\n    id\n    repositoryPath\n    branch\n    moduleDirectory\n    tagRegex\n    globPatterns\n    autoSpeculativePlan\n    webhookDisabled\n    vcsProvider {\n      id\n      name\n      description\n      type\n      autoCreateWebhooks\n    }\n  }\n}\n\nfragment VCSProviderListFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment VCSProvidersFragment_group on Group {\n  ...VCSProviderListFragment_group\n  ...NewVCSProviderFragment_group\n  ...EditVCSProviderFragment_group\n  ...VCSProviderDetailsFragment_group\n  ...EditVCSProviderOAuthCredentialsFragment_group\n}\n\nfragment VCSWorkspaceLinkSourceFragment_workspace on Workspace {\n  workspaceVcsProviderLink {\n    branch\n    id\n  }\n}\n\nfragment VariableListItemFragment_variable on NamespaceVariable {\n  id\n  key\n  category\n  sensitive\n  value\n  namespacePath\n  latestVersionId\n  metadata {\n    updatedAt\n  }\n}\n\nfragment VariablesFragment_variables on Namespace {\n  __isNamespace: __typename\n  id\n  fullPath\n  variables {\n    id\n    key\n    category\n    ...VariableListItemFragment_variable\n  }\n}\n\nfragment WorkspaceAdvancedSettingsDeleteDialogFragment_workspace on Workspace {\n  name\n  fullPath\n}\n\nfragment WorkspaceAdvancedSettingsFragment_workspace on Workspace {\n  name\n  fullPath\n  ...WorkspaceAdvancedSettingsDeleteDialogFragment_workspace\n  ...MigrateWorkspaceDialogFragment_workspace\n}\n\nfragment WorkspaceDetailsCurrentJobFragment_workspace on Workspace {\n  id\n  fullPath\n  currentJob {\n    id\n    type\n    run {\n      id\n      status\n      createdBy\n      isDestroy\n      moduleSource\n      moduleVersion\n      metadata {\n        createdAt\n      }\n      configurationVersion {\n        id\n      }\n      plan {\n        status\n        metadata {\n          createdAt\n        }\n        id\n      }\n      apply {\n        status\n        triggeredBy\n        metadata {\n          createdAt\n          updatedAt\n        }\n        id\n      }\n    }\n  }\n}\n\nfragment WorkspaceDetailsDriftDetectionFragment_workspace on Workspace {\n  id\n  fullPath\n  assessment {\n    hasDrift\n    startedAt\n    completedAt\n    run {\n      id\n    }\n    id\n  }\n}\n\nfragment WorkspaceDetailsEmptyFragment_workspace on Workspace {\n  id\n  fullPath\n}\n\nfragment WorkspaceDetailsFragment_workspace on Workspace {\n  id\n  name\n  description\n  fullPath\n  ...WorkspaceDetailsIndexFragment_workspace\n  ...AssignedManagedIdentityListFragment_assignedManagedIdentities\n  ...RunsFragment_runs\n  ...StateVersionsFragment_stateVersions\n  ...VariablesFragment_variables\n  ...NamespaceMembershipsFragment_memberships\n  ...WorkspaceSettingsFragment_workspace\n  ...NamespaceActivityFragment_activity\n}\n\nfragment WorkspaceDetailsIndexFragment_workspace on Workspace {\n  id\n  name\n  description\n  fullPath\n  preventDestroyPlan\n  metadata {\n    trn\n  }\n  assessment {\n    hasDrift\n    id\n  }\n  ...WorkspaceDetailsEmptyFragment_workspace\n  ...WorkspaceDetailsCurrentJobFragment_workspace\n  ...WorkspaceNotificationPreferenceFragment_workspace\n  currentJob {\n    id\n  }\n  currentStateVersion {\n    id\n    ...StateVersionOutputsFragment_outputs\n    ...StateVersionResourcesFragment_resources\n    ...StateVersionDependenciesFragment_dependencies\n    ...StateVersionFileFragment_stateVersion\n    metadata {\n      createdAt\n    }\n    run {\n      ...StateVersionInputVariablesFragment_variables\n      id\n      status\n      createdBy\n      isDestroy\n      moduleSource\n      moduleVersion\n      metadata {\n        createdAt\n      }\n      configurationVersion {\n        id\n        vcsEvent {\n          status\n          id\n        }\n      }\n      plan {\n        status\n        metadata {\n          createdAt\n        }\n        id\n      }\n      apply {\n        status\n        triggeredBy\n        metadata {\n          createdAt\n          updatedAt\n        }\n        id\n      }\n    }\n  }\n  ...WorkspaceDetailsDriftDetectionFragment_workspace\n}\n\nfragment WorkspaceDriftDetectionSettingsFragment_workspace on Workspace {\n  fullPath\n  driftDetectionEnabled {\n    inherited\n    value\n    ...DriftDetectionSettingsFormFragment_driftDetectionEnabled\n  }\n}\n\nfragment WorkspaceGeneralSettingsFragment_workspace on Workspace {\n  name\n  description\n  fullPath\n}\n\nfragment WorkspaceListFragment_group on Group {\n  id\n}\n\nfragment WorkspaceNotificationPreferenceFragment_workspace on Workspace {\n  fullPath\n}\n\nfragment WorkspaceRunSettingsFragment_workspace on Workspace {\n  name\n  description\n  fullPath\n  maxJobDuration\n  terraformVersion\n  preventDestroyPlan\n  ...MaxJobDurationSettingFragment_workspace\n}\n\nfragment WorkspaceRunnerSettingsFragment_workspace on Workspace {\n  fullPath\n  runnerTags {\n    inherited\n    namespacePath\n    value\n    ...RunnerSettingsForm_runnerTags\n  }\n}\n\nfragment WorkspaceSettingsFragment_workspace on Workspace {\n  name\n  description\n  fullPath\n  ...WorkspaceGeneralSettingsFragment_workspace\n  ...WorkspaceRunnerSettingsFragment_workspace\n  ...WorkspaceRunSettingsFragment_workspace\n  ...WorkspaceDriftDetectionSettingsFragment_workspace\n  ...WorkspaceAdvancedSettingsFragment_workspace\n  ...WorkspaceVCSProviderSettingsFragment_workspace\n  ...WorkspaceStateSettingsFragment_workspace\n}\n\nfragment WorkspaceStateSettingsFragment_workspace on Workspace {\n  fullPath\n  locked\n}\n\nfragment WorkspaceVCSProviderSettingsFragment_workspace on Workspace {\n  workspaceVcsProviderLink {\n    id\n  }\n  fullPath\n  groupPath\n  vcsProviders(first: 10, includeInherited: true) {\n    edges {\n      node {\n        id\n      }\n    }\n  }\n  ...EditVCSProviderLinkFragment_workspace\n  ...NewVCSProviderLinkFragment_workspace\n}\n"
  }
};
})();

(node as any).hash = "bf01becf9164b83ab43327a67729eb3d";

export default node;
