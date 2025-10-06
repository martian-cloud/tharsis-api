/**
 * @generated SignedSource<<39fed0f6feb7282d471fdbee7bd22a23>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type WorkspaceSubscriptionInput = {
  workspaceId?: string | null | undefined;
  workspacePath?: string | null | undefined;
};
export type WorkspaceDetailsWorkspaceEventSubscription$variables = {
  input: WorkspaceSubscriptionInput;
};
export type WorkspaceDetailsWorkspaceEventSubscription$data = {
  readonly workspaceEvents: {
    readonly action: string;
    readonly workspace: {
      readonly id: string;
      readonly " $fragmentSpreads": FragmentRefs<"WorkspaceDetailsIndexFragment_workspace">;
    };
  };
};
export type WorkspaceDetailsWorkspaceEventSubscription = {
  response: WorkspaceDetailsWorkspaceEventSubscription$data;
  variables: WorkspaceDetailsWorkspaceEventSubscription$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "input"
  }
],
v1 = [
  {
    "kind": "Variable",
    "name": "input",
    "variableName": "input"
  }
],
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "action",
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
  "name": "name",
  "storageKey": null
},
v5 = [
  (v3/*: any*/)
],
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "type",
  "storageKey": null
},
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "status",
  "storageKey": null
},
v8 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "createdBy",
  "storageKey": null
},
v9 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "isDestroy",
  "storageKey": null
},
v10 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "moduleSource",
  "storageKey": null
},
v11 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "moduleVersion",
  "storageKey": null
},
v12 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "createdAt",
  "storageKey": null
},
v13 = {
  "alias": null,
  "args": null,
  "concreteType": "ResourceMetadata",
  "kind": "LinkedField",
  "name": "metadata",
  "plural": false,
  "selections": [
    (v12/*: any*/)
  ],
  "storageKey": null
},
v14 = {
  "alias": null,
  "args": null,
  "concreteType": "Plan",
  "kind": "LinkedField",
  "name": "plan",
  "plural": false,
  "selections": [
    (v7/*: any*/),
    (v13/*: any*/),
    (v3/*: any*/)
  ],
  "storageKey": null
},
v15 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "updatedAt",
  "storageKey": null
},
v16 = {
  "alias": null,
  "args": null,
  "concreteType": "Apply",
  "kind": "LinkedField",
  "name": "apply",
  "plural": false,
  "selections": [
    (v7/*: any*/),
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
        (v12/*: any*/),
        (v15/*: any*/)
      ],
      "storageKey": null
    },
    (v3/*: any*/)
  ],
  "storageKey": null
},
v17 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "value",
  "storageKey": null
},
v18 = {
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
    "name": "WorkspaceDetailsWorkspaceEventSubscription",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "WorkspaceEvent",
        "kind": "LinkedField",
        "name": "workspaceEvents",
        "plural": false,
        "selections": [
          (v2/*: any*/),
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
                "args": null,
                "kind": "FragmentSpread",
                "name": "WorkspaceDetailsIndexFragment_workspace"
              }
            ],
            "storageKey": null
          }
        ],
        "storageKey": null
      }
    ],
    "type": "Subscription",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "WorkspaceDetailsWorkspaceEventSubscription",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "WorkspaceEvent",
        "kind": "LinkedField",
        "name": "workspaceEvents",
        "plural": false,
        "selections": [
          (v2/*: any*/),
          {
            "alias": null,
            "args": null,
            "concreteType": "Workspace",
            "kind": "LinkedField",
            "name": "workspace",
            "plural": false,
            "selections": [
              (v3/*: any*/),
              (v4/*: any*/),
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
                    "selections": (v5/*: any*/),
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
                  (v6/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "Run",
                    "kind": "LinkedField",
                    "name": "run",
                    "plural": false,
                    "selections": [
                      (v3/*: any*/),
                      (v7/*: any*/),
                      (v8/*: any*/),
                      (v9/*: any*/),
                      (v10/*: any*/),
                      (v11/*: any*/),
                      (v13/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "ConfigurationVersion",
                        "kind": "LinkedField",
                        "name": "configurationVersion",
                        "plural": false,
                        "selections": (v5/*: any*/),
                        "storageKey": null
                      },
                      (v14/*: any*/),
                      (v16/*: any*/)
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
                      (v4/*: any*/),
                      (v17/*: any*/),
                      (v6/*: any*/),
                      (v18/*: any*/),
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
                      (v4/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "provider",
                        "storageKey": null
                      },
                      (v6/*: any*/),
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
                              (v15/*: any*/)
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
                            "selections": (v5/*: any*/),
                            "storageKey": null
                          }
                        ],
                        "storageKey": null
                      }
                    ],
                    "storageKey": null
                  },
                  (v13/*: any*/),
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
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "key",
                            "storageKey": null
                          },
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
                          (v17/*: any*/),
                          (v18/*: any*/),
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
                      (v7/*: any*/),
                      (v8/*: any*/),
                      (v9/*: any*/),
                      (v10/*: any*/),
                      (v11/*: any*/),
                      (v13/*: any*/),
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
                              (v7/*: any*/),
                              (v3/*: any*/)
                            ],
                            "storageKey": null
                          }
                        ],
                        "storageKey": null
                      },
                      (v14/*: any*/),
                      (v16/*: any*/)
                    ],
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
      }
    ]
  },
  "params": {
    "cacheID": "cb85e728abc4ba1a0929dc04b1067ede",
    "id": null,
    "metadata": {},
    "name": "WorkspaceDetailsWorkspaceEventSubscription",
    "operationKind": "subscription",
    "text": "subscription WorkspaceDetailsWorkspaceEventSubscription(\n  $input: WorkspaceSubscriptionInput!\n) {\n  workspaceEvents(input: $input) {\n    action\n    workspace {\n      id\n      ...WorkspaceDetailsIndexFragment_workspace\n    }\n  }\n}\n\nfragment StateVersionDependenciesFragment_dependencies on StateVersion {\n  dependencies {\n    workspacePath\n    ...StateVersionDependencyListItemFragment_dependency\n  }\n}\n\nfragment StateVersionDependencyListItemFragment_dependency on StateVersionDependency {\n  workspacePath\n  stateVersion {\n    id\n    metadata {\n      updatedAt\n    }\n  }\n  workspace {\n    id\n    currentStateVersion {\n      id\n    }\n  }\n}\n\nfragment StateVersionFileFragment_stateVersion on StateVersion {\n  id\n}\n\nfragment StateVersionInputVariableListItemFragment_variable on RunVariable {\n  key\n  value\n  category\n  namespacePath\n  sensitive\n  versionId\n  includedInTfConfig\n}\n\nfragment StateVersionInputVariablesFragment_variables on Run {\n  variables {\n    key\n    category\n    namespacePath\n    includedInTfConfig\n    ...StateVersionInputVariableListItemFragment_variable\n  }\n}\n\nfragment StateVersionOutputListItemFragment_output on StateVersionOutput {\n  name\n  value\n  type\n  sensitive\n}\n\nfragment StateVersionOutputsFragment_outputs on StateVersion {\n  outputs {\n    name\n    ...StateVersionOutputListItemFragment_output\n    id\n  }\n}\n\nfragment StateVersionResourceListItemFragment_resource on StateVersionResource {\n  name\n  type\n  provider\n  mode\n  module\n}\n\nfragment StateVersionResourcesFragment_resources on StateVersion {\n  resources {\n    name\n    provider\n    type\n    ...StateVersionResourceListItemFragment_resource\n  }\n}\n\nfragment WorkspaceDetailsCurrentJobFragment_workspace on Workspace {\n  id\n  fullPath\n  currentJob {\n    id\n    type\n    run {\n      id\n      status\n      createdBy\n      isDestroy\n      moduleSource\n      moduleVersion\n      metadata {\n        createdAt\n      }\n      configurationVersion {\n        id\n      }\n      plan {\n        status\n        metadata {\n          createdAt\n        }\n        id\n      }\n      apply {\n        status\n        triggeredBy\n        metadata {\n          createdAt\n          updatedAt\n        }\n        id\n      }\n    }\n  }\n}\n\nfragment WorkspaceDetailsDriftDetectionFragment_workspace on Workspace {\n  id\n  fullPath\n  assessment {\n    hasDrift\n    startedAt\n    completedAt\n    run {\n      id\n    }\n    id\n  }\n}\n\nfragment WorkspaceDetailsEmptyFragment_workspace on Workspace {\n  id\n  fullPath\n}\n\nfragment WorkspaceDetailsIndexFragment_workspace on Workspace {\n  id\n  name\n  description\n  fullPath\n  preventDestroyPlan\n  metadata {\n    trn\n  }\n  assessment {\n    hasDrift\n    id\n  }\n  ...WorkspaceDetailsEmptyFragment_workspace\n  ...WorkspaceDetailsCurrentJobFragment_workspace\n  ...WorkspaceNotificationPreferenceFragment_workspace\n  currentJob {\n    id\n  }\n  currentStateVersion {\n    id\n    ...StateVersionOutputsFragment_outputs\n    ...StateVersionResourcesFragment_resources\n    ...StateVersionDependenciesFragment_dependencies\n    ...StateVersionFileFragment_stateVersion\n    metadata {\n      createdAt\n    }\n    run {\n      ...StateVersionInputVariablesFragment_variables\n      id\n      status\n      createdBy\n      isDestroy\n      moduleSource\n      moduleVersion\n      metadata {\n        createdAt\n      }\n      configurationVersion {\n        id\n        vcsEvent {\n          status\n          id\n        }\n      }\n      plan {\n        status\n        metadata {\n          createdAt\n        }\n        id\n      }\n      apply {\n        status\n        triggeredBy\n        metadata {\n          createdAt\n          updatedAt\n        }\n        id\n      }\n    }\n  }\n  ...WorkspaceDetailsDriftDetectionFragment_workspace\n}\n\nfragment WorkspaceNotificationPreferenceFragment_workspace on Workspace {\n  fullPath\n}\n"
  }
};
})();

(node as any).hash = "aa773485d469bef3223351f2dbb228d4";

export default node;
