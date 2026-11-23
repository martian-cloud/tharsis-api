/**
 * @generated SignedSource<<cec46a5757494f22d889d8c560eb401b>>
 * @lightSyntaxTransform
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
    readonly workspace: {
      readonly " $fragmentSpreads": FragmentRefs<"WorkspaceSettingsContentFragment_workspace">;
    } | null | undefined;
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
  "name": "inherited",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "namespacePath",
  "storageKey": null
},
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "value",
  "storageKey": null
},
v7 = [
  (v4/*:: as any*/),
  (v6/*:: as any*/),
  (v5/*:: as any*/)
],
v8 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "groupPath",
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*:: as any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "WorkspaceSettingsQuery",
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
                {
                  "args": null,
                  "kind": "FragmentSpread",
                  "name": "WorkspaceSettingsContentFragment_workspace"
                }
              ],
              "type": "Workspace",
              "abstractKey": null
            },
            "kind": "AliasedInlineFragmentSpread",
            "name": "workspace"
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
    "name": "WorkspaceSettingsQuery",
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
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "__typename",
            "storageKey": null
          },
          (v2/*:: as any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              (v3/*:: as any*/),
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
                "concreteType": "NamespaceRunnerTags",
                "kind": "LinkedField",
                "name": "runnerTags",
                "plural": false,
                "selections": [
                  (v4/*:: as any*/),
                  (v5/*:: as any*/),
                  (v6/*:: as any*/)
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
                "selections": (v7/*:: as any*/),
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "NamespaceProviderMirrorEnabled",
                "kind": "LinkedField",
                "name": "providerMirrorEnabled",
                "plural": false,
                "selections": (v7/*:: as any*/),
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "NamespaceOutputVisibility",
                "kind": "LinkedField",
                "name": "outputVisibility",
                "plural": false,
                "selections": (v7/*:: as any*/),
                "storageKey": null
              },
              (v8/*:: as any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "WorkspaceVCSProviderLink",
                "kind": "LinkedField",
                "name": "workspaceVcsProviderLink",
                "plural": false,
                "selections": [
                  (v2/*:: as any*/),
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
                    "name": "branch",
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
                    "concreteType": "VCSProvider",
                    "kind": "LinkedField",
                    "name": "vcsProvider",
                    "plural": false,
                    "selections": [
                      (v2/*:: as any*/),
                      (v3/*:: as any*/),
                      (v8/*:: as any*/),
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
                        "name": "autoCreateWebhooks",
                        "storageKey": null
                      }
                    ],
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "usesLegacyWebhookToken",
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
                    "value": 1
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
                          (v2/*:: as any*/)
                        ],
                        "storageKey": null
                      }
                    ],
                    "storageKey": null
                  }
                ],
                "storageKey": "vcsProviders(first:1,includeInherited:true)"
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
                  (v6/*:: as any*/)
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
    "cacheID": "c3c350b3b7e762e11ca55ed992c4e72c",
    "id": null,
    "metadata": {},
    "name": "WorkspaceSettingsQuery",
    "operationKind": "query",
    "text": "query WorkspaceSettingsQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ... on Workspace {\n      ...WorkspaceSettingsContentFragment_workspace\n    }\n    id\n  }\n}\n\nfragment DriftDetectionSettingsFormFragment_driftDetectionEnabled on NamespaceDriftDetectionEnabled {\n  inherited\n  namespacePath\n  value\n}\n\nfragment EditVCSProviderLinkFragment_link on WorkspaceVCSProviderLink {\n  id\n  repositoryPath\n  branch\n  moduleDirectory\n  tagRegex\n  globPatterns\n  autoSpeculativePlan\n  webhookDisabled\n  vcsProvider {\n    id\n    name\n    groupPath\n    type\n    autoCreateWebhooks\n  }\n}\n\nfragment MaxJobDurationSettingFragment_workspace on Workspace {\n  maxJobDuration\n}\n\nfragment MigrateWorkspaceDialogFragment_workspace on Workspace {\n  name\n  fullPath\n  groupPath\n}\n\nfragment OutputVisibilitySettingsFormFragment_outputVisibility on NamespaceOutputVisibility {\n  inherited\n  namespacePath\n  value\n}\n\nfragment ProviderMirrorSettingsFormFragment_providerMirrorEnabled on NamespaceProviderMirrorEnabled {\n  inherited\n  namespacePath\n  value\n}\n\nfragment RunnerSettingsForm_runnerTags on NamespaceRunnerTags {\n  inherited\n  namespacePath\n  value\n}\n\nfragment VCSProviderLinkDangerZoneFragment_link on WorkspaceVCSProviderLink {\n  id\n  repositoryPath\n  vcsProvider {\n    autoCreateWebhooks\n    id\n  }\n}\n\nfragment VCSProviderLinkWebhookFragment_link on WorkspaceVCSProviderLink {\n  id\n  repositoryPath\n  webhookDisabled\n  usesLegacyWebhookToken\n  vcsProvider {\n    type\n    autoCreateWebhooks\n    id\n  }\n}\n\nfragment WorkspaceAdvancedSettingsFragment_workspace on Workspace {\n  name\n  fullPath\n  ...MigrateWorkspaceDialogFragment_workspace\n}\n\nfragment WorkspaceDriftDetectionSettingsFragment_workspace on Workspace {\n  fullPath\n  driftDetectionEnabled {\n    inherited\n    value\n    ...DriftDetectionSettingsFormFragment_driftDetectionEnabled\n  }\n}\n\nfragment WorkspaceGeneralSettingsFragment_workspace on Workspace {\n  name\n  description\n  fullPath\n}\n\nfragment WorkspaceLabelSettingsFragment_workspace on Workspace {\n  id\n  fullPath\n  description\n  labels {\n    key\n    value\n  }\n}\n\nfragment WorkspaceOutputVisibilitySettingsFragment_workspace on Workspace {\n  fullPath\n  outputVisibility {\n    inherited\n    value\n    ...OutputVisibilitySettingsFormFragment_outputVisibility\n  }\n}\n\nfragment WorkspaceProviderMirrorSettingsFragment_workspace on Workspace {\n  fullPath\n  providerMirrorEnabled {\n    inherited\n    value\n    ...ProviderMirrorSettingsFormFragment_providerMirrorEnabled\n  }\n}\n\nfragment WorkspaceRunSettingsFragment_workspace on Workspace {\n  name\n  description\n  fullPath\n  maxJobDuration\n  terraformVersion\n  preventDestroyPlan\n  ...MaxJobDurationSettingFragment_workspace\n}\n\nfragment WorkspaceRunnerSettingsFragment_workspace on Workspace {\n  fullPath\n  runnerTags {\n    inherited\n    namespacePath\n    value\n    ...RunnerSettingsForm_runnerTags\n  }\n}\n\nfragment WorkspaceSettingsContentFragment_workspace on Workspace {\n  name\n  description\n  fullPath\n  ...WorkspaceGeneralSettingsFragment_workspace\n  ...WorkspaceRunnerSettingsFragment_workspace\n  ...WorkspaceRunSettingsFragment_workspace\n  ...WorkspaceDriftDetectionSettingsFragment_workspace\n  ...WorkspaceProviderMirrorSettingsFragment_workspace\n  ...WorkspaceOutputVisibilitySettingsFragment_workspace\n  ...WorkspaceAdvancedSettingsFragment_workspace\n  ...WorkspaceVCSProviderSettingsFragment_workspace\n  ...WorkspaceStateSettingsFragment_workspace\n  ...WorkspaceLabelSettingsFragment_workspace\n}\n\nfragment WorkspaceStateSettingsFragment_workspace on Workspace {\n  fullPath\n  locked\n}\n\nfragment WorkspaceVCSProviderSettingsFragment_workspace on Workspace {\n  fullPath\n  groupPath\n  workspaceVcsProviderLink {\n    id\n    ...EditVCSProviderLinkFragment_link\n    ...VCSProviderLinkWebhookFragment_link\n    ...VCSProviderLinkDangerZoneFragment_link\n  }\n  vcsProviders(first: 1, includeInherited: true) {\n    edges {\n      node {\n        id\n      }\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "645c707678f254c97767f182e50ed7b3";

export default node;
