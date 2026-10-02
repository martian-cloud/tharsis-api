/**
 * @generated SignedSource<<4a69cb7abb6069832dce391c4e1a70a7>>
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
  "concreteType": "ResourceMetadata",
  "kind": "LinkedField",
  "name": "metadata",
  "plural": false,
  "selections": [
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
v8 = [
  {
    "kind": "Literal",
    "name": "first",
    "value": 0
  }
],
v9 = [
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "totalCount",
    "storageKey": null
  }
],
v10 = {
  "kind": "TypeDiscriminator",
  "abstractKey": "__isNamespace"
},
v11 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "key",
  "storageKey": null
},
v12 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "value",
  "storageKey": null
},
v13 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "status",
  "storageKey": null
},
v14 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "hasAdvisoryFailures",
  "storageKey": null
},
v15 = [
  (v13/*: any*/),
  (v3/*: any*/)
],
v16 = {
  "alias": null,
  "args": null,
  "concreteType": "RunTaskStage",
  "kind": "LinkedField",
  "name": "taskStages",
  "plural": true,
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "stageName",
      "storageKey": null
    },
    (v13/*: any*/)
  ],
  "storageKey": null
},
v17 = {
  "alias": null,
  "args": null,
  "concreteType": "Apply",
  "kind": "LinkedField",
  "name": "apply",
  "plural": false,
  "selections": (v15/*: any*/),
  "storageKey": null
},
v18 = {
  "alias": null,
  "args": null,
  "concreteType": "Workspace",
  "kind": "LinkedField",
  "name": "workspace",
  "plural": false,
  "selections": [
    (v4/*: any*/),
    (v3/*: any*/)
  ],
  "storageKey": null
},
v19 = {
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
v20 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "type",
  "storageKey": null
},
v21 = [
  (v3/*: any*/)
],
v22 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "sensitive",
  "storageKey": null
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
              (v7/*: any*/),
              {
                "alias": null,
                "args": (v8/*: any*/),
                "concreteType": "WorkspaceConnection",
                "kind": "LinkedField",
                "name": "workspaces",
                "plural": false,
                "selections": (v9/*: any*/),
                "storageKey": "workspaces(first:0)"
              },
              {
                "alias": null,
                "args": (v8/*: any*/),
                "concreteType": "GroupConnection",
                "kind": "LinkedField",
                "name": "descendentGroups",
                "plural": false,
                "selections": (v9/*: any*/),
                "storageKey": "descendentGroups(first:0)"
              },
              (v10/*: any*/)
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
                "name": "locked",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "destroyed",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "preventDestroyPlan",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "WorkspaceLabel",
                "kind": "LinkedField",
                "name": "labels",
                "plural": true,
                "selections": [
                  (v11/*: any*/),
                  (v12/*: any*/)
                ],
                "storageKey": null
              },
              (v7/*: any*/),
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
                    "selections": [
                      (v3/*: any*/),
                      (v13/*: any*/)
                    ],
                    "storageKey": null
                  }
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "Run",
                "kind": "LinkedField",
                "name": "currentApplyRun",
                "plural": false,
                "selections": [
                  (v3/*: any*/),
                  (v13/*: any*/),
                  (v14/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "Plan",
                    "kind": "LinkedField",
                    "name": "plan",
                    "plural": false,
                    "selections": (v15/*: any*/),
                    "storageKey": null
                  },
                  (v16/*: any*/),
                  (v17/*: any*/),
                  (v18/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "isDestroy",
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
                  (v19/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "StateVersionInventory",
                    "kind": "LinkedField",
                    "name": "inventory",
                    "plural": false,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "StateVersionResource",
                        "kind": "LinkedField",
                        "name": "resources",
                        "plural": true,
                        "selections": [
                          (v2/*: any*/),
                          (v5/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "provider",
                            "storageKey": null
                          },
                          (v20/*: any*/),
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
                              }
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
                                "selections": (v21/*: any*/),
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
                        "args": null,
                        "concreteType": "TerraformCheckResult",
                        "kind": "LinkedField",
                        "name": "checkResults",
                        "plural": true,
                        "selections": [
                          (v5/*: any*/),
                          (v13/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": "TerraformCheckResultObject",
                            "kind": "LinkedField",
                            "name": "objects",
                            "plural": true,
                            "selections": [
                              {
                                "alias": null,
                                "args": null,
                                "kind": "ScalarField",
                                "name": "address",
                                "storageKey": null
                              },
                              (v13/*: any*/),
                              {
                                "alias": null,
                                "args": null,
                                "kind": "ScalarField",
                                "name": "failureMessages",
                                "storageKey": null
                              }
                            ],
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
                    "args": null,
                    "concreteType": "Run",
                    "kind": "LinkedField",
                    "name": "run",
                    "plural": false,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "moduleSource",
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "moduleVersion",
                        "storageKey": null
                      },
                      (v3/*: any*/),
                      (v13/*: any*/),
                      (v14/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "Plan",
                        "kind": "LinkedField",
                        "name": "plan",
                        "plural": false,
                        "selections": [
                          (v13/*: any*/),
                          (v3/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": "PlanSummary",
                            "kind": "LinkedField",
                            "name": "summary",
                            "plural": false,
                            "selections": [
                              {
                                "alias": null,
                                "args": null,
                                "kind": "ScalarField",
                                "name": "resourceAdditions",
                                "storageKey": null
                              },
                              {
                                "alias": null,
                                "args": null,
                                "kind": "ScalarField",
                                "name": "resourceChanges",
                                "storageKey": null
                              },
                              {
                                "alias": null,
                                "args": null,
                                "kind": "ScalarField",
                                "name": "resourceDestructions",
                                "storageKey": null
                              }
                            ],
                            "storageKey": null
                          }
                        ],
                        "storageKey": null
                      },
                      (v16/*: any*/),
                      (v17/*: any*/),
                      (v18/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "ConfigurationVersion",
                        "kind": "LinkedField",
                        "name": "configurationVersion",
                        "plural": false,
                        "selections": [
                          (v3/*: any*/),
                          (v19/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": "VCSEvent",
                            "kind": "LinkedField",
                            "name": "vcsEvent",
                            "plural": false,
                            "selections": (v21/*: any*/),
                            "storageKey": null
                          }
                        ],
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "RunVariable",
                        "kind": "LinkedField",
                        "name": "variables",
                        "plural": true,
                        "selections": [
                          (v11/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "category",
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
                            "name": "includedInTfConfig",
                            "storageKey": null
                          },
                          (v12/*: any*/),
                          (v22/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "versionId",
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
                    "args": null,
                    "concreteType": "StateVersionOutput",
                    "kind": "LinkedField",
                    "name": "outputs",
                    "plural": true,
                    "selections": [
                      (v5/*: any*/),
                      (v12/*: any*/),
                      (v20/*: any*/),
                      (v22/*: any*/),
                      (v3/*: any*/)
                    ],
                    "storageKey": null
                  }
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
                  }
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "groupPath",
                "storageKey": null
              },
              (v10/*: any*/)
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
    "cacheID": "02c3634007a8f715d97a199830443aab",
    "id": null,
    "metadata": {},
    "name": "GroupOrWorkspaceRendererQuery",
    "operationKind": "query",
    "text": "query GroupOrWorkspaceRendererQuery(\n  $fullPath: String!\n) {\n  namespace(fullPath: $fullPath) {\n    __typename\n    id\n    fullPath\n    ... on Group {\n      ...GroupDetailsFragment_group\n    }\n    ... on Workspace {\n      ...WorkspaceDetailsFragment_workspace\n    }\n  }\n}\n\nfragment AssignedManagedIdentityListFragment_workspace on Workspace {\n  id\n  fullPath\n}\n\nfragment AssignedPolicyListFragment_workspace on Workspace {\n  id\n  fullPath\n}\n\nfragment CleanupPoliciesFragment_namespace on Namespace {\n  __isNamespace: __typename\n  fullPath\n}\n\nfragment ConfigurationVersionDetailsFragment_workspace on Workspace {\n  fullPath\n}\n\nfragment CreateRunFragment_workspace on Workspace {\n  id\n  fullPath\n  workspaceVcsProviderLink {\n    id\n  }\n  ...ModuleSourceFragment_workspace\n  ...VCSWorkspaceLinkSourceFragment_workspace\n}\n\nfragment EditFederatedRegistryFragment_group on Group {\n  fullPath\n}\n\nfragment EditGroupRunnerFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment EditManagedIdentityFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment EditServiceAccountFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment EditTerraformModuleFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment EditVCSProviderFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment EditVCSProviderOAuthCredentialsFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment FederatedRegistriesFragment_group on Group {\n  ...FederatedRegistryListFragment_group\n  ...FederatedRegistryDetailsFragment_group\n  ...NewFederatedRegistryFragment_group\n  ...EditFederatedRegistryFragment_group\n}\n\nfragment FederatedRegistryDetailsFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment FederatedRegistryListFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment GPGKeyListFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment GPGKeysFragment_group on Group {\n  ...GPGKeyListFragment_group\n  ...NewGPGKeyFragment_group\n}\n\nfragment GroupDetailsFragment_group on Group {\n  id\n  fullPath\n  name\n  ...GroupDetailsIndexFragment_group\n  ...ManagedIdentitiesFragment_group\n  ...GroupRunnersFragment_group\n  ...GroupRunsFragment_group\n  ...ServiceAccountsFragment_group\n  ...TerraformModulesFragment_group\n  ...GroupPackagesFragment_group\n  ...GroupPoliciesFragment_group\n  ...VCSProvidersFragment_group\n  ...FederatedRegistriesFragment_group\n  ...VariablesFragment_variables\n  ...NamespaceMembershipsFragment_memberships\n  ...GPGKeysFragment_group\n  ...ProviderMirrorsFragment_namespace\n  ...CleanupPoliciesFragment_namespace\n  ...NamespaceActivityFragment_activity\n  ...GroupSettingsFragment_group\n}\n\nfragment GroupDetailsIndexFragment_group on Group {\n  id\n  name\n  description\n  fullPath\n  metadata {\n    trn\n  }\n  workspaces(first: 0) {\n    totalCount\n  }\n  descendentGroups(first: 0) {\n    totalCount\n  }\n  ...WorkspaceListFragment_group\n  ...MigrateGroupDialogFragment_group\n  ...GroupNotificationPreferenceFragment_group\n}\n\nfragment GroupEditPackageFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment GroupNewPackageFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment GroupNotificationPreferenceFragment_group on Group {\n  fullPath\n}\n\nfragment GroupPackageListFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment GroupPackagesFragment_group on Group {\n  ...GroupPackageListFragment_group\n  ...GroupNewPackageFragment_group\n  ...GroupEditPackageFragment_group\n}\n\nfragment GroupPoliciesFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment GroupRunnerDetailsFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment GroupRunnersFragment_group on Group {\n  ...GroupRunnersListFragment_group\n  ...NewGroupRunnerFragment_group\n  ...EditGroupRunnerFragment_group\n  ...GroupRunnerDetailsFragment_group\n}\n\nfragment GroupRunnersListFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment GroupRunsFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment GroupSettingsFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment ManagedIdentitiesFragment_group on Group {\n  ...ManagedIdentityListFragment_group\n  ...NewManagedIdentityFragment_group\n  ...EditManagedIdentityFragment_group\n  ...ManagedIdentityDetailsFragment_group\n}\n\nfragment ManagedIdentityDetailsFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment ManagedIdentityListFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment MigrateGroupDialogFragment_group on Group {\n  name\n  fullPath\n}\n\nfragment ModuleSourceFragment_workspace on Workspace {\n  fullPath\n}\n\nfragment ModuleSourceLinkFragment_run on Run {\n  moduleSource\n  moduleVersion\n}\n\nfragment NamespaceActivityFragment_activity on Namespace {\n  __isNamespace: __typename\n  __typename\n  fullPath\n}\n\nfragment NamespaceMembershipListFragment_memberships on Namespace {\n  __isNamespace: __typename\n  id\n  fullPath\n}\n\nfragment NamespaceMembershipsFragment_memberships on Namespace {\n  __isNamespace: __typename\n  ...NamespaceMembershipsIndexFragment_memberships\n  ...NewNamespaceMembershipFragment_memberships\n}\n\nfragment NamespaceMembershipsIndexFragment_memberships on Namespace {\n  __isNamespace: __typename\n  fullPath\n  ...NamespaceMembershipListFragment_memberships\n}\n\nfragment NewFederatedRegistryFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment NewGPGKeyFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment NewGroupRunnerFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment NewManagedIdentityFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment NewNamespaceMembershipFragment_memberships on Namespace {\n  __isNamespace: __typename\n  fullPath\n}\n\nfragment NewServiceAccountFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment NewVCSProviderFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment ProviderMirrorsFragment_namespace on Namespace {\n  __isNamespace: __typename\n  fullPath\n}\n\nfragment RunDetailsFragment_details on Workspace {\n  id\n  fullPath\n}\n\nfragment RunStageIconsFragment_run on Run {\n  id\n  status\n  hasAdvisoryFailures\n  plan {\n    status\n    id\n  }\n  taskStages {\n    stageName\n    status\n  }\n  apply {\n    status\n    id\n  }\n  workspace {\n    fullPath\n    id\n  }\n}\n\nfragment RunsFragment_runs on Workspace {\n  fullPath\n  ...RunsIndexFragment_runs\n  ...CreateRunFragment_workspace\n  ...RunDetailsFragment_details\n}\n\nfragment RunsIndexFragment_runs on Workspace {\n  id\n  fullPath\n}\n\nfragment ServiceAccountDetailsFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment ServiceAccountListFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment ServiceAccountsFragment_group on Group {\n  ...ServiceAccountListFragment_group\n  ...ServiceAccountDetailsFragment_group\n  ...NewServiceAccountFragment_group\n  ...EditServiceAccountFragment_group\n}\n\nfragment StateVersionCheckResultRowFragment_checkResult on TerraformCheckResult {\n  name\n  status\n  objects {\n    address\n    status\n    failureMessages\n  }\n}\n\nfragment StateVersionCheckResultsFragment_checkResults on StateVersionInventory {\n  checkResults {\n    name\n    status\n    ...StateVersionCheckResultRowFragment_checkResult\n  }\n}\n\nfragment StateVersionDependenciesFragment_dependencies on StateVersionInventory {\n  dependencies {\n    workspacePath\n    ...StateVersionDependencyListItemFragment_dependency\n  }\n}\n\nfragment StateVersionDependencyListItemFragment_dependency on StateVersionDependency {\n  workspacePath\n  stateVersion {\n    id\n    metadata {\n      updatedAt\n    }\n  }\n  workspace {\n    id\n    currentStateVersion {\n      id\n    }\n  }\n}\n\nfragment StateVersionDetailsFragment_details on Workspace {\n  id\n  fullPath\n}\n\nfragment StateVersionFileFragment_stateVersion on StateVersion {\n  id\n}\n\nfragment StateVersionInputVariableListItemFragment_variable on RunVariable {\n  key\n  value\n  category\n  namespacePath\n  sensitive\n  versionId\n  includedInTfConfig\n}\n\nfragment StateVersionInputVariablesFragment_variables on Run {\n  variables {\n    key\n    category\n    namespacePath\n    includedInTfConfig\n    ...StateVersionInputVariableListItemFragment_variable\n  }\n}\n\nfragment StateVersionListFragment_workspace on Workspace {\n  id\n  fullPath\n}\n\nfragment StateVersionOutputListItemFragment_output on StateVersionOutput {\n  name\n  value\n  type\n  sensitive\n}\n\nfragment StateVersionOutputsFragment_outputs on StateVersion {\n  outputs {\n    name\n    ...StateVersionOutputListItemFragment_output\n    id\n  }\n}\n\nfragment StateVersionResourceListItemFragment_resource on StateVersionResource {\n  name\n  type\n  provider\n  mode\n  module\n}\n\nfragment StateVersionResourcesFragment_resources on StateVersionInventory {\n  resources {\n    name\n    provider\n    type\n    ...StateVersionResourceListItemFragment_resource\n  }\n}\n\nfragment StateVersionsFragment_stateVersions on Workspace {\n  fullPath\n  ...StateVersionListFragment_workspace\n  ...StateVersionDetailsFragment_details\n}\n\nfragment TerraformModuleListFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment TerraformModulesFragment_group on Group {\n  ...TerraformModuleListFragment_group\n  ...EditTerraformModuleFragment_group\n}\n\nfragment VCSProviderDetailsFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment VCSProviderListFragment_group on Group {\n  id\n  fullPath\n}\n\nfragment VCSProvidersFragment_group on Group {\n  ...VCSProviderListFragment_group\n  ...NewVCSProviderFragment_group\n  ...EditVCSProviderFragment_group\n  ...VCSProviderDetailsFragment_group\n  ...EditVCSProviderOAuthCredentialsFragment_group\n}\n\nfragment VCSWorkspaceLinkSourceFragment_workspace on Workspace {\n  workspaceVcsProviderLink {\n    branch\n    id\n  }\n}\n\nfragment VariablesFragment_variables on Namespace {\n  __isNamespace: __typename\n  id\n  fullPath\n}\n\nfragment WorkspaceDetailsDriftDetectionFragment_workspace on Workspace {\n  id\n  fullPath\n  assessment {\n    hasDrift\n    startedAt\n    completedAt\n    run {\n      id\n      status\n    }\n    id\n  }\n}\n\nfragment WorkspaceDetailsEmptyFragment_workspace on Workspace {\n  id\n  fullPath\n}\n\nfragment WorkspaceDetailsFragment_workspace on Workspace {\n  id\n  name\n  description\n  fullPath\n  ...WorkspaceDetailsIndexFragment_workspace\n  ...AssignedPolicyListFragment_workspace\n  ...AssignedManagedIdentityListFragment_workspace\n  ...RunsFragment_runs\n  ...ConfigurationVersionDetailsFragment_workspace\n  ...StateVersionsFragment_stateVersions\n  ...VariablesFragment_variables\n  ...NamespaceMembershipsFragment_memberships\n  ...WorkspaceSettingsFragment_workspace\n  ...NamespaceActivityFragment_activity\n  ...ProviderMirrorsFragment_namespace\n  ...CleanupPoliciesFragment_namespace\n  ...WorkspaceRoleBindingFragment_workspace\n}\n\nfragment WorkspaceDetailsIndexFragment_workspace on Workspace {\n  id\n  name\n  description\n  fullPath\n  locked\n  destroyed\n  preventDestroyPlan\n  labels {\n    key\n    value\n  }\n  metadata {\n    trn\n  }\n  assessment {\n    hasDrift\n    id\n  }\n  ...WorkspaceDetailsEmptyFragment_workspace\n  ...WorkspaceDetailsStatusPanelFragment_workspace\n  ...WorkspaceNotificationPreferenceFragment_workspace\n  currentApplyRun {\n    id\n  }\n  currentStateVersion {\n    id\n    ...StateVersionOutputsFragment_outputs\n    inventory {\n      ...StateVersionResourcesFragment_resources\n      ...StateVersionDependenciesFragment_dependencies\n      ...StateVersionCheckResultsFragment_checkResults\n    }\n    ...StateVersionFileFragment_stateVersion\n    run {\n      id\n      ...StateVersionInputVariablesFragment_variables\n    }\n  }\n  ...WorkspaceDetailsDriftDetectionFragment_workspace\n}\n\nfragment WorkspaceDetailsStatusPanelFragment_workspace on Workspace {\n  fullPath\n  currentApplyRun {\n    ...RunStageIconsFragment_run\n    id\n    isDestroy\n  }\n  currentStateVersion {\n    id\n    metadata {\n      createdAt\n    }\n    inventory {\n      resources {\n        __typename\n      }\n    }\n    run {\n      ...ModuleSourceLinkFragment_run\n      ...RunStageIconsFragment_run\n      id\n      status\n      apply {\n        status\n        id\n      }\n      plan {\n        summary {\n          resourceAdditions\n          resourceChanges\n          resourceDestructions\n        }\n        id\n      }\n      moduleSource\n      moduleVersion\n      configurationVersion {\n        id\n        metadata {\n          createdAt\n        }\n        vcsEvent {\n          id\n        }\n      }\n    }\n  }\n}\n\nfragment WorkspaceListFragment_group on Group {\n  id\n}\n\nfragment WorkspaceNotificationPreferenceFragment_workspace on Workspace {\n  fullPath\n}\n\nfragment WorkspaceRoleBindingFragment_workspace on Workspace {\n  id\n  name\n  fullPath\n  groupPath\n}\n\nfragment WorkspaceSettingsFragment_workspace on Workspace {\n  id\n  fullPath\n}\n"
  }
};
})();

(node as any).hash = "bf01becf9164b83ab43327a67729eb3d";

export default node;
