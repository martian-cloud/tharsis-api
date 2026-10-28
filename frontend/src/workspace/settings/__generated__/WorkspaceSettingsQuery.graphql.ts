/**
 * @generated SignedSource<<e18bad9fba9e0583572fe269e02b5eed>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type WorkspaceSettingsQuery$variables = {
  id: string;
};
export type WorkspaceSettingsQuery$data = {
  readonly node: {
    readonly " $fragmentSpreads": FragmentRefs<"WorkspaceSettingsContentFragment_workspace">;
  } | null | undefined;
};
export type WorkspaceSettingsQuery = {
  response: WorkspaceSettingsQuery$data;
  variables: WorkspaceSettingsQuery$variables;
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
  "name": "name",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "description",
  "storageKey": null
},
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "inherited",
  "storageKey": null
},
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "namespacePath",
  "storageKey": null
},
v8 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "value",
  "storageKey": null
},
v9 = [
  (v6/*: any*/),
  (v8/*: any*/),
  (v7/*: any*/)
],
v10 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "type",
  "storageKey": null
},
v11 = [
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
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "WorkspaceSettingsQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
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
                "name": "WorkspaceSettingsContentFragment_workspace"
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
    "name": "WorkspaceSettingsQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          (v2/*: any*/),
          (v3/*: any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              (v4/*: any*/),
              (v5/*: any*/),
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
                "concreteType": "NamespaceRunnerTags",
                "kind": "LinkedField",
                "name": "runnerTags",
                "plural": false,
                "selections": [
                  (v6/*: any*/),
                  (v7/*: any*/),
                  (v8/*: any*/)
                ],
                "storageKey": null
              },
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
                "concreteType": "NamespaceDriftDetectionEnabled",
                "kind": "LinkedField",
                "name": "driftDetectionEnabled",
                "plural": false,
                "selections": (v9/*: any*/),
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "NamespaceProviderMirrorEnabled",
                "kind": "LinkedField",
                "name": "providerMirrorEnabled",
                "plural": false,
                "selections": (v9/*: any*/),
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "NamespaceOutputVisibility",
                "kind": "LinkedField",
                "name": "outputVisibility",
                "plural": false,
                "selections": (v9/*: any*/),
                "storageKey": null
              },
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
                "concreteType": "WorkspaceVCSProviderLink",
                "kind": "LinkedField",
                "name": "workspaceVcsProviderLink",
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
                    "name": "createdBy",
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
                      (v2/*: any*/),
                      {
                        "kind": "TypeDiscriminator",
                        "abstractKey": "__isActor"
                      },
                      (v10/*: any*/),
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
                              (v3/*: any*/)
                            ],
                            "storageKey": null
                          }
                        ],
                        "type": "UserActor",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v11/*: any*/),
                        "type": "ServiceAccountActor",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v11/*: any*/),
                        "type": "VCSProviderActor",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v11/*: any*/),
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
                    "name": "branch",
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
                      (v4/*: any*/),
                      (v5/*: any*/),
                      (v10/*: any*/),
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
              {
                "alias": null,
                "args": [
                  {
                    "kind": "Literal",
                    "name": "first",
                    "value": 10
                  },
                  {
                    "kind": "Literal",
                    "name": "includeInherited",
                    "value": true
                  }
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
                        "selections": [
                          (v3/*: any*/)
                        ],
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
              {
                "alias": null,
                "args": null,
                "concreteType": "WorkspaceLabel",
                "kind": "LinkedField",
                "name": "labels",
                "plural": true,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "key",
                    "storageKey": null
                  },
                  (v8/*: any*/)
                ],
                "storageKey": null
              }
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
    "cacheID": "064ced8d6d02e0bb6409d63d74590659",
    "id": null,
    "metadata": {},
    "name": "WorkspaceSettingsQuery",
    "operationKind": "query",
    "text": "query WorkspaceSettingsQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ... on Workspace {\n      ...WorkspaceSettingsContentFragment_workspace\n    }\n    id\n  }\n}\n\nfragment ActorDisplayFragment_actor on Actor {\n  __isActor: __typename\n  type\n  principalId\n  ... on UserActor {\n    user {\n      email\n      id\n    }\n  }\n  ... on ServiceAccountActor {\n    trnSnapshot\n  }\n  ... on VCSProviderActor {\n    trnSnapshot\n  }\n  ... on JobActor {\n    trnSnapshot\n  }\n}\n\nfragment DriftDetectionSettingsFormFragment_driftDetectionEnabled on NamespaceDriftDetectionEnabled {\n  inherited\n  namespacePath\n  value\n}\n\nfragment EditVCSProviderLinkFragment_workspace on Workspace {\n  fullPath\n  workspaceVcsProviderLink {\n    id\n    metadata {\n      createdAt\n    }\n    createdBy\n    createdByActor {\n      __typename\n      ...ActorDisplayFragment_actor\n    }\n    repositoryPath\n    autoSpeculativePlan\n    webhookDisabled\n    moduleDirectory\n    branch\n    tagRegex\n    globPatterns\n    vcsProvider {\n      id\n      name\n      description\n      type\n      autoCreateWebhooks\n    }\n  }\n  ...VCSProviderLinkFormFragment_workspace\n}\n\nfragment MaxJobDurationSettingFragment_workspace on Workspace {\n  maxJobDuration\n}\n\nfragment MigrateWorkspaceDialogFragment_workspace on Workspace {\n  name\n  fullPath\n  groupPath\n}\n\nfragment NewVCSProviderLinkFragment_workspace on Workspace {\n  fullPath\n  ...VCSProviderLinkFormFragment_workspace\n}\n\nfragment OutputVisibilitySettingsFormFragment_outputVisibility on NamespaceOutputVisibility {\n  inherited\n  namespacePath\n  value\n}\n\nfragment ProviderMirrorSettingsFormFragment_providerMirrorEnabled on NamespaceProviderMirrorEnabled {\n  inherited\n  namespacePath\n  value\n}\n\nfragment RunnerSettingsForm_runnerTags on NamespaceRunnerTags {\n  inherited\n  namespacePath\n  value\n}\n\nfragment VCSProviderLinkFormFragment_workspace on Workspace {\n  fullPath\n  workspaceVcsProviderLink {\n    id\n    repositoryPath\n    branch\n    moduleDirectory\n    tagRegex\n    globPatterns\n    autoSpeculativePlan\n    webhookDisabled\n    vcsProvider {\n      id\n      name\n      description\n      type\n      autoCreateWebhooks\n    }\n  }\n}\n\nfragment WorkspaceAdvancedSettingsFragment_workspace on Workspace {\n  name\n  fullPath\n  ...MigrateWorkspaceDialogFragment_workspace\n}\n\nfragment WorkspaceDriftDetectionSettingsFragment_workspace on Workspace {\n  fullPath\n  driftDetectionEnabled {\n    inherited\n    value\n    ...DriftDetectionSettingsFormFragment_driftDetectionEnabled\n  }\n}\n\nfragment WorkspaceGeneralSettingsFragment_workspace on Workspace {\n  name\n  description\n  fullPath\n}\n\nfragment WorkspaceLabelSettingsFragment_workspace on Workspace {\n  id\n  fullPath\n  description\n  labels {\n    key\n    value\n  }\n}\n\nfragment WorkspaceOutputVisibilitySettingsFragment_workspace on Workspace {\n  fullPath\n  outputVisibility {\n    inherited\n    value\n    ...OutputVisibilitySettingsFormFragment_outputVisibility\n  }\n}\n\nfragment WorkspaceProviderMirrorSettingsFragment_workspace on Workspace {\n  fullPath\n  providerMirrorEnabled {\n    inherited\n    value\n    ...ProviderMirrorSettingsFormFragment_providerMirrorEnabled\n  }\n}\n\nfragment WorkspaceRunSettingsFragment_workspace on Workspace {\n  name\n  description\n  fullPath\n  maxJobDuration\n  terraformVersion\n  preventDestroyPlan\n  ...MaxJobDurationSettingFragment_workspace\n}\n\nfragment WorkspaceRunnerSettingsFragment_workspace on Workspace {\n  fullPath\n  runnerTags {\n    inherited\n    namespacePath\n    value\n    ...RunnerSettingsForm_runnerTags\n  }\n}\n\nfragment WorkspaceSettingsContentFragment_workspace on Workspace {\n  name\n  description\n  fullPath\n  ...WorkspaceGeneralSettingsFragment_workspace\n  ...WorkspaceRunnerSettingsFragment_workspace\n  ...WorkspaceRunSettingsFragment_workspace\n  ...WorkspaceDriftDetectionSettingsFragment_workspace\n  ...WorkspaceProviderMirrorSettingsFragment_workspace\n  ...WorkspaceOutputVisibilitySettingsFragment_workspace\n  ...WorkspaceAdvancedSettingsFragment_workspace\n  ...WorkspaceVCSProviderSettingsFragment_workspace\n  ...WorkspaceStateSettingsFragment_workspace\n  ...WorkspaceLabelSettingsFragment_workspace\n}\n\nfragment WorkspaceStateSettingsFragment_workspace on Workspace {\n  fullPath\n  locked\n}\n\nfragment WorkspaceVCSProviderSettingsFragment_workspace on Workspace {\n  workspaceVcsProviderLink {\n    id\n  }\n  fullPath\n  groupPath\n  vcsProviders(first: 10, includeInherited: true) {\n    edges {\n      node {\n        id\n      }\n    }\n  }\n  ...EditVCSProviderLinkFragment_workspace\n  ...NewVCSProviderLinkFragment_workspace\n}\n"
  }
};
})();

(node as any).hash = "b19e9230031be8ad458f1ac22d0b859a";

export default node;
