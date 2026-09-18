/**
 * @generated SignedSource<<a53d89d03c7bac5da4852c452085f0cd>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type NamespaceActivityQuery$variables = {
  after?: string | null | undefined;
  before?: string | null | undefined;
  first?: number | null | undefined;
  last?: number | null | undefined;
  namespacePath?: string | null | undefined;
};
export type NamespaceActivityQuery$data = {
  readonly " $fragmentSpreads": FragmentRefs<"NamespaceActivityConnectionFragment_activity">;
};
export type NamespaceActivityQuery = {
  response: NamespaceActivityQuery$data;
  variables: NamespaceActivityQuery$variables;
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
  "name": "last"
},
v4 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "namespacePath"
},
v5 = [
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
    "name": "includeNested",
    "value": true
  },
  {
    "kind": "Variable",
    "name": "last",
    "variableName": "last"
  },
  {
    "kind": "Variable",
    "name": "namespacePath",
    "variableName": "namespacePath"
  },
  {
    "kind": "Literal",
    "name": "sort",
    "value": "CREATED_DESC"
  }
],
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
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
  "name": "name",
  "storageKey": null
},
v9 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "fullPath",
  "storageKey": null
},
v10 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "description",
  "storageKey": null
},
v11 = [
  (v8/*: any*/),
  (v9/*: any*/),
  (v10/*: any*/)
],
v12 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "resourcePath",
  "storageKey": null
},
v13 = [
  (v8/*: any*/),
  (v10/*: any*/),
  (v12/*: any*/)
],
v14 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "username",
  "storageKey": null
},
v15 = {
  "kind": "InlineFragment",
  "selections": [
    (v14/*: any*/)
  ],
  "type": "User",
  "abstractKey": null
},
v16 = [
  (v8/*: any*/)
],
v17 = {
  "kind": "InlineFragment",
  "selections": (v16/*: any*/),
  "type": "Team",
  "abstractKey": null
},
v18 = {
  "kind": "InlineFragment",
  "selections": [
    (v12/*: any*/)
  ],
  "type": "ServiceAccount",
  "abstractKey": null
},
v19 = [
  (v6/*: any*/)
],
v20 = {
  "kind": "InlineFragment",
  "selections": (v19/*: any*/),
  "type": "Node",
  "abstractKey": "__isNode"
},
v21 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "key",
  "storageKey": null
},
v22 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "registryNamespace",
  "storageKey": null
},
v23 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "version",
  "storageKey": null
},
v24 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "system",
  "storageKey": null
},
v25 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "groupPath",
  "storageKey": null
},
v26 = [
  (v8/*: any*/),
  (v25/*: any*/)
],
v27 = {
  "alias": null,
  "args": null,
  "concreteType": null,
  "kind": "LinkedField",
  "name": "member",
  "plural": false,
  "selections": [
    (v7/*: any*/),
    (v15/*: any*/),
    (v18/*: any*/),
    (v17/*: any*/),
    (v20/*: any*/)
  ],
  "storageKey": null
},
v28 = [
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "previousGroupPath",
    "storageKey": null
  }
],
v29 = [
  (v21/*: any*/),
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "value",
    "storageKey": null
  }
],
v30 = [
  {
    "alias": null,
    "args": null,
    "concreteType": "LabelChangePayload",
    "kind": "LinkedField",
    "name": "labelChanges",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "WorkspaceLabel",
        "kind": "LinkedField",
        "name": "added",
        "plural": true,
        "selections": (v29/*: any*/),
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "concreteType": "WorkspaceLabel",
        "kind": "LinkedField",
        "name": "updated",
        "plural": true,
        "selections": (v29/*: any*/),
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "removed",
        "storageKey": null
      }
    ],
    "storageKey": null
  }
],
v31 = {
  "alias": null,
  "args": null,
  "concreteType": "User",
  "kind": "LinkedField",
  "name": "user",
  "plural": false,
  "selections": [
    (v14/*: any*/),
    (v6/*: any*/)
  ],
  "storageKey": null
},
v32 = [
  (v31/*: any*/),
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "maintainer",
    "storageKey": null
  }
];
return {
  "fragment": {
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/),
      (v2/*: any*/),
      (v3/*: any*/),
      (v4/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "NamespaceActivityQuery",
    "selections": [
      {
        "args": null,
        "kind": "FragmentSpread",
        "name": "NamespaceActivityConnectionFragment_activity"
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [
      (v2/*: any*/),
      (v3/*: any*/),
      (v0/*: any*/),
      (v1/*: any*/),
      (v4/*: any*/)
    ],
    "kind": "Operation",
    "name": "NamespaceActivityQuery",
    "selections": [
      {
        "alias": null,
        "args": (v5/*: any*/),
        "concreteType": "ActivityEventConnection",
        "kind": "LinkedField",
        "name": "activityEvents",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "ActivityEventEdge",
            "kind": "LinkedField",
            "name": "edges",
            "plural": true,
            "selections": [
              {
                "alias": null,
                "args": null,
                "concreteType": "ActivityEvent",
                "kind": "LinkedField",
                "name": "node",
                "plural": false,
                "selections": [
                  (v6/*: any*/),
                  (v7/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": null,
                    "kind": "LinkedField",
                    "name": "target",
                    "plural": false,
                    "selections": [
                      (v7/*: any*/),
                      (v6/*: any*/),
                      {
                        "kind": "InlineFragment",
                        "selections": (v11/*: any*/),
                        "type": "Workspace",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v11/*: any*/),
                        "type": "Group",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v13/*: any*/),
                        "type": "ManagedIdentity",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": null,
                            "kind": "LinkedField",
                            "name": "member",
                            "plural": false,
                            "selections": [
                              (v7/*: any*/),
                              (v15/*: any*/),
                              (v17/*: any*/),
                              (v18/*: any*/),
                              (v20/*: any*/)
                            ],
                            "storageKey": null
                          }
                        ],
                        "type": "NamespaceMembership",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "kind",
                            "storageKey": null
                          }
                        ],
                        "type": "CleanupPolicy",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "gpgKeyId",
                            "storageKey": null
                          }
                        ],
                        "type": "GPGKey",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "runStage",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": "ManagedIdentity",
                            "kind": "LinkedField",
                            "name": "managedIdentity",
                            "plural": false,
                            "selections": [
                              (v6/*: any*/),
                              (v12/*: any*/)
                            ],
                            "storageKey": null
                          }
                        ],
                        "type": "ManagedIdentityAccessRule",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v13/*: any*/),
                        "type": "ServiceAccount",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          (v21/*: any*/)
                        ],
                        "type": "NamespaceVariable",
                        "abstractKey": null
                      },
                      (v17/*: any*/),
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          (v8/*: any*/),
                          (v22/*: any*/)
                        ],
                        "type": "TerraformProvider",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          (v23/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": "TerraformProvider",
                            "kind": "LinkedField",
                            "name": "provider",
                            "plural": false,
                            "selections": [
                              (v8/*: any*/),
                              (v22/*: any*/),
                              (v6/*: any*/)
                            ],
                            "storageKey": null
                          }
                        ],
                        "type": "TerraformProviderVersion",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          (v8/*: any*/),
                          (v24/*: any*/),
                          (v22/*: any*/)
                        ],
                        "type": "TerraformModule",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          (v23/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": "TerraformModule",
                            "kind": "LinkedField",
                            "name": "module",
                            "plural": false,
                            "selections": [
                              (v8/*: any*/),
                              (v24/*: any*/),
                              (v22/*: any*/),
                              (v6/*: any*/)
                            ],
                            "storageKey": null
                          }
                        ],
                        "type": "TerraformModuleVersion",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v16/*: any*/),
                        "type": "VCSProvider",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v16/*: any*/),
                        "type": "Role",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v16/*: any*/),
                        "type": "Runner",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "hostname",
                            "storageKey": null
                          }
                        ],
                        "type": "FederatedRegistry",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          (v23/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "providerAddress",
                            "storageKey": null
                          }
                        ],
                        "type": "TerraformProviderVersionMirror",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": "Run",
                            "kind": "LinkedField",
                            "name": "run",
                            "plural": false,
                            "selections": (v19/*: any*/),
                            "storageKey": null
                          }
                        ],
                        "type": "RunGate",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v26/*: any*/),
                        "type": "Package",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          (v23/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": "Package",
                            "kind": "LinkedField",
                            "name": "package",
                            "plural": false,
                            "selections": [
                              (v6/*: any*/),
                              (v8/*: any*/),
                              (v25/*: any*/)
                            ],
                            "storageKey": null
                          }
                        ],
                        "type": "PackageVersion",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v26/*: any*/),
                        "type": "Policy",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": "Workspace",
                            "kind": "LinkedField",
                            "name": "workspace",
                            "plural": false,
                            "selections": [
                              (v8/*: any*/),
                              (v9/*: any*/),
                              (v6/*: any*/)
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
                            "selections": [
                              (v8/*: any*/),
                              (v6/*: any*/)
                            ],
                            "storageKey": null
                          }
                        ],
                        "type": "WorkspaceRoleBinding",
                        "abstractKey": null
                      }
                    ],
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "action",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": null,
                    "kind": "LinkedField",
                    "name": "payload",
                    "plural": false,
                    "selections": [
                      (v7/*: any*/),
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          (v8/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "type",
                            "storageKey": null
                          }
                        ],
                        "type": "ActivityEventDeleteChildResourcePayload",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          (v27/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "role",
                            "storageKey": null
                          }
                        ],
                        "type": "ActivityEventCreateNamespaceMembershipPayload",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          (v27/*: any*/)
                        ],
                        "type": "ActivityEventRemoveNamespaceMembershipPayload",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v28/*: any*/),
                        "type": "ActivityEventMigrateWorkspacePayload",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": "WorkspaceLabel",
                            "kind": "LinkedField",
                            "name": "labels",
                            "plural": true,
                            "selections": (v29/*: any*/),
                            "storageKey": null
                          }
                        ],
                        "type": "ActivityEventCreateWorkspacePayload",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v30/*: any*/),
                        "type": "ActivityEventUpdateWorkspacePayload",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v28/*: any*/),
                        "type": "ActivityEventMigrateGroupPayload",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v28/*: any*/),
                        "type": "ActivityEventMoveManagedIdentityPayload",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "prevRole",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "newRole",
                            "storageKey": null
                          }
                        ],
                        "type": "ActivityEventUpdateNamespaceMembershipPayload",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          {
                            "alias": "runUpdateType",
                            "args": null,
                            "kind": "ScalarField",
                            "name": "type",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "nodePath",
                            "storageKey": null
                          }
                        ],
                        "type": "ActivityEventUpdateRunPayload",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v32/*: any*/),
                        "type": "ActivityEventUpdateTeamMemberPayload",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          (v31/*: any*/)
                        ],
                        "type": "ActivityEventRemoveTeamMemberPayload",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v32/*: any*/),
                        "type": "ActivityEventAddTeamMemberPayload",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": "TerraformModuleLabel",
                            "kind": "LinkedField",
                            "name": "labels",
                            "plural": true,
                            "selections": (v29/*: any*/),
                            "storageKey": null
                          }
                        ],
                        "type": "ActivityEventCreateTerraformModulePayload",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v30/*: any*/),
                        "type": "ActivityEventUpdateTerraformModulePayload",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          {
                            "alias": "gateUpdateType",
                            "args": null,
                            "kind": "ScalarField",
                            "name": "type",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "comment",
                            "storageKey": null
                          }
                        ],
                        "type": "ActivityEventUpdateRunGatePayload",
                        "abstractKey": null
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
                    "name": "initiator",
                    "plural": false,
                    "selections": [
                      (v7/*: any*/),
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          (v14/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "email",
                            "storageKey": null
                          }
                        ],
                        "type": "User",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          (v8/*: any*/),
                          (v12/*: any*/)
                        ],
                        "type": "ServiceAccount",
                        "abstractKey": null
                      },
                      (v20/*: any*/)
                    ],
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "namespacePath",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "targetType",
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
        "args": (v5/*: any*/),
        "filters": [
          "namespacePath",
          "includeNested",
          "sort"
        ],
        "handle": "connection",
        "key": "NamespaceActivity_activityEvents",
        "kind": "LinkedHandle",
        "name": "activityEvents"
      }
    ]
  },
  "params": {
    "cacheID": "9af626490ec67b0f820d6fd245fd415c",
    "id": null,
    "metadata": {},
    "name": "NamespaceActivityQuery",
    "operationKind": "query",
    "text": "query NamespaceActivityQuery(\n  $first: Int\n  $last: Int\n  $after: String\n  $before: String\n  $namespacePath: String\n) {\n  ...NamespaceActivityConnectionFragment_activity\n}\n\nfragment ActivityEventCleanupPolicyTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on CleanupPolicy {\n      kind\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventFederatedRegistryTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on FederatedRegistry {\n      id\n      hostname\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventGPGKeyTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on GPGKey {\n      id\n      gpgKeyId\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventGroupTargetFragment_event on ActivityEvent {\n  action\n  target {\n    __typename\n    ... on Group {\n      name\n      fullPath\n      description\n    }\n    id\n  }\n  payload {\n    __typename\n    ... on ActivityEventDeleteChildResourcePayload {\n      name\n      type\n    }\n    ... on ActivityEventCreateNamespaceMembershipPayload {\n      member {\n        __typename\n        ... on User {\n          username\n        }\n        ... on ServiceAccount {\n          resourcePath\n        }\n        ... on Team {\n          name\n        }\n        ... on Node {\n          __isNode: __typename\n          id\n        }\n      }\n      role\n    }\n    ... on ActivityEventRemoveNamespaceMembershipPayload {\n      member {\n        __typename\n        ... on User {\n          username\n        }\n        ... on ServiceAccount {\n          resourcePath\n        }\n        ... on Team {\n          name\n        }\n        ... on Node {\n          __isNode: __typename\n          id\n        }\n      }\n    }\n    ... on ActivityEventMigrateGroupPayload {\n      previousGroupPath\n    }\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventListFragment_connection on ActivityEventConnection {\n  edges {\n    node {\n      id\n      target {\n        __typename\n        id\n      }\n      ...ActivityEventWorkspaceTargetFragment_event\n      ...ActivityEventGroupTargetFragment_event\n      ...ActivityEventManagedIdentityTargetFragment_event\n      ...ActivityEventNamespaceMembershipTargetFragment_event\n      ...ActivityEventCleanupPolicyTargetFragment_event\n      ...ActivityEventGPGKeyTargetFragment_event\n      ...ActivityEventManagedIdentityAccessRuleTargetFragment_event\n      ...ActivityEventServiceAccountTargetFragment_event\n      ...ActivityEventVariableTargetFragment_event\n      ...ActivityEventRunTargetFragment_event\n      ...ActivityEventStateVersionTargetFragment_event\n      ...ActivityEventTeamTargetFragment_event\n      ...ActivityEventTerraformProviderTargetFragment_event\n      ...ActivityEventTerraformProviderVersionTargetFragment_event\n      ...ActivityEventTerraformModuleTargetFragment_event\n      ...ActivityEventTerraformModuleVersionTargetFragment_event\n      ...ActivityEventVCSProviderTargetFragment_event\n      ...ActivityEventRoleTargetFragment_event\n      ...ActivityEventRunnerTargetFragment_event\n      ...ActivityEventFederatedRegistryTargetFragment_event\n      ...ActivityEventTerraformProviderVersionMirrorTargetFragment_event\n      ...ActivityEventRunGateTargetFragment_event\n      ...ActivityEventPackageTargetFragment_event\n      ...ActivityEventPackageVersionTargetFragment_event\n      ...ActivityEventPolicyTargetFragment_event\n      ...ActivityEventWorkspaceRoleBindingTargetFragment_event\n      ...ActivityEventTargetNotFoundFragment_event\n    }\n  }\n}\n\nfragment ActivityEventListItemFragment_event on ActivityEvent {\n  metadata {\n    createdAt\n  }\n  id\n  initiator {\n    __typename\n    ... on User {\n      username\n      email\n    }\n    ... on ServiceAccount {\n      name\n      resourcePath\n    }\n    ... on Node {\n      __isNode: __typename\n      id\n    }\n  }\n}\n\nfragment ActivityEventManagedIdentityAccessRuleTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on ManagedIdentityAccessRule {\n      runStage\n      managedIdentity {\n        id\n        resourcePath\n      }\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventManagedIdentityTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on ManagedIdentity {\n      id\n      name\n      description\n      resourcePath\n    }\n    id\n  }\n  payload {\n    __typename\n    ... on ActivityEventMoveManagedIdentityPayload {\n      previousGroupPath\n    }\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventNamespaceMembershipTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on NamespaceMembership {\n      member {\n        __typename\n        ... on User {\n          username\n        }\n        ... on Team {\n          name\n        }\n        ... on ServiceAccount {\n          resourcePath\n        }\n        ... on Node {\n          __isNode: __typename\n          id\n        }\n      }\n    }\n    id\n  }\n  payload {\n    __typename\n    ... on ActivityEventUpdateNamespaceMembershipPayload {\n      prevRole\n      newRole\n    }\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventPackageTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on Package {\n      id\n      name\n      groupPath\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventPackageVersionTargetFragment_event on ActivityEvent {\n  action\n  target {\n    __typename\n    ... on PackageVersion {\n      id\n      version\n      package {\n        id\n        name\n        groupPath\n      }\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventPolicyTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on Policy {\n      id\n      name\n      groupPath\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventRoleTargetFragment_event on ActivityEvent {\n  action\n  target {\n    __typename\n    ... on Role {\n      name\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventRunGateTargetFragment_event on ActivityEvent {\n  namespacePath\n  target {\n    __typename\n    ... on RunGate {\n      id\n      run {\n        id\n      }\n    }\n    id\n  }\n  payload {\n    __typename\n    ... on ActivityEventUpdateRunGatePayload {\n      gateUpdateType: type\n      comment\n    }\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventRunTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on Run {\n      id\n    }\n    id\n  }\n  payload {\n    __typename\n    ... on ActivityEventUpdateRunPayload {\n      runUpdateType: type\n      nodePath\n    }\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventRunnerTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on Runner {\n      name\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventServiceAccountTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on ServiceAccount {\n      id\n      name\n      description\n      resourcePath\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventStateVersionTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on StateVersion {\n      id\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventTargetNotFoundFragment_event on ActivityEvent {\n  targetType\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventTeamTargetFragment_event on ActivityEvent {\n  action\n  target {\n    __typename\n    ... on Team {\n      name\n    }\n    id\n  }\n  payload {\n    __typename\n    ... on ActivityEventUpdateTeamMemberPayload {\n      user {\n        username\n        id\n      }\n      maintainer\n    }\n    ... on ActivityEventRemoveTeamMemberPayload {\n      user {\n        username\n        id\n      }\n    }\n    ... on ActivityEventAddTeamMemberPayload {\n      user {\n        username\n        id\n      }\n      maintainer\n    }\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventTerraformModuleTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on TerraformModule {\n      name\n      system\n      registryNamespace\n    }\n    id\n  }\n  payload {\n    __typename\n    ... on ActivityEventCreateTerraformModulePayload {\n      labels {\n        key\n        value\n      }\n    }\n    ... on ActivityEventUpdateTerraformModulePayload {\n      labelChanges {\n        added {\n          key\n          value\n        }\n        updated {\n          key\n          value\n        }\n        removed\n      }\n    }\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventTerraformModuleVersionTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on TerraformModuleVersion {\n      version\n      module {\n        name\n        system\n        registryNamespace\n        id\n      }\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventTerraformProviderTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on TerraformProvider {\n      name\n      registryNamespace\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventTerraformProviderVersionMirrorTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on TerraformProviderVersionMirror {\n      id\n      version\n      providerAddress\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventTerraformProviderVersionTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on TerraformProviderVersion {\n      version\n      provider {\n        name\n        registryNamespace\n        id\n      }\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventVCSProviderTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on VCSProvider {\n      name\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventVariableTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on NamespaceVariable {\n      key\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventWorkspaceRoleBindingTargetFragment_event on ActivityEvent {\n  action\n  target {\n    __typename\n    ... on WorkspaceRoleBinding {\n      workspace {\n        name\n        fullPath\n        id\n      }\n      role {\n        name\n        id\n      }\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventWorkspaceTargetFragment_event on ActivityEvent {\n  action\n  target {\n    __typename\n    ... on Workspace {\n      name\n      fullPath\n      description\n    }\n    id\n  }\n  payload {\n    __typename\n    ... on ActivityEventDeleteChildResourcePayload {\n      name\n      type\n    }\n    ... on ActivityEventCreateNamespaceMembershipPayload {\n      member {\n        __typename\n        ... on User {\n          username\n        }\n        ... on ServiceAccount {\n          resourcePath\n        }\n        ... on Team {\n          name\n        }\n        ... on Node {\n          __isNode: __typename\n          id\n        }\n      }\n      role\n    }\n    ... on ActivityEventRemoveNamespaceMembershipPayload {\n      member {\n        __typename\n        ... on User {\n          username\n        }\n        ... on ServiceAccount {\n          resourcePath\n        }\n        ... on Team {\n          name\n        }\n        ... on Node {\n          __isNode: __typename\n          id\n        }\n      }\n    }\n    ... on ActivityEventMigrateWorkspacePayload {\n      previousGroupPath\n    }\n    ... on ActivityEventCreateWorkspacePayload {\n      labels {\n        key\n        value\n      }\n    }\n    ... on ActivityEventUpdateWorkspacePayload {\n      labelChanges {\n        added {\n          key\n          value\n        }\n        updated {\n          key\n          value\n        }\n        removed\n      }\n    }\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment NamespaceActivityConnectionFragment_activity on Query {\n  activityEvents(after: $after, before: $before, first: $first, last: $last, namespacePath: $namespacePath, includeNested: true, sort: CREATED_DESC) {\n    edges {\n      node {\n        id\n        __typename\n      }\n      cursor\n    }\n    ...ActivityEventListFragment_connection\n    pageInfo {\n      endCursor\n      hasNextPage\n      hasPreviousPage\n      startCursor\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "947695372016feb7b8409573224739a2";

export default node;
