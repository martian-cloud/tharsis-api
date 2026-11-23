/**
 * @generated SignedSource<<68dfe51388d6a74de17f899743962ec0>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type VCSProviderType = "github" | "gitlab" | "%future added value";
export type CreateWorkspaceVCSProviderLinkInput = {
  autoSpeculativePlan: boolean;
  branch?: string | null | undefined;
  clientMutationId?: string | null | undefined;
  globPatterns: ReadonlyArray<string>;
  moduleDirectory?: string | null | undefined;
  providerId: string;
  repositoryPath: string;
  tagRegex?: string | null | undefined;
  webhookDisabled: boolean;
  workspaceId?: string | null | undefined;
  workspacePath?: string | null | undefined;
};
export type NewVCSProviderLinkMutation$variables = {
  input: CreateWorkspaceVCSProviderLinkInput;
};
export type NewVCSProviderLinkMutation$data = {
  readonly createWorkspaceVCSProviderLink: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly vcsProviderLink: {
      readonly id: string;
      readonly repositoryPath: string;
      readonly vcsProvider: {
        readonly autoCreateWebhooks: boolean;
        readonly type: VCSProviderType;
      };
      readonly workspace: {
        readonly id: string;
        readonly workspaceVcsProviderLink: {
          readonly id: string;
          readonly " $fragmentSpreads": FragmentRefs<"EditVCSProviderLinkFragment_link" | "VCSProviderLinkDangerZoneFragment_link" | "VCSProviderLinkWebhookFragment_link">;
        } | null | undefined;
      };
    } | null | undefined;
    readonly webhookToken: string | null | undefined;
    readonly webhookUrl: string | null | undefined;
  };
};
export type NewVCSProviderLinkMutation = {
  response: NewVCSProviderLinkMutation$data;
  variables: NewVCSProviderLinkMutation$variables;
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
  "name": "id",
  "storageKey": null
},
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "repositoryPath",
  "storageKey": null
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "type",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "autoCreateWebhooks",
  "storageKey": null
},
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "webhookToken",
  "storageKey": null
},
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "webhookUrl",
  "storageKey": null
},
v8 = {
  "alias": null,
  "args": null,
  "concreteType": "Problem",
  "kind": "LinkedField",
  "name": "problems",
  "plural": true,
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "message",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "field",
      "storageKey": null
    },
    (v4/*:: as any*/)
  ],
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*:: as any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "NewVCSProviderLinkMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*:: as any*/),
        "concreteType": "CreateWorkspaceVCSProviderLinkPayload",
        "kind": "LinkedField",
        "name": "createWorkspaceVCSProviderLink",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "WorkspaceVCSProviderLink",
            "kind": "LinkedField",
            "name": "vcsProviderLink",
            "plural": false,
            "selections": [
              (v2/*:: as any*/),
              (v3/*:: as any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "VCSProvider",
                "kind": "LinkedField",
                "name": "vcsProvider",
                "plural": false,
                "selections": [
                  (v4/*:: as any*/),
                  (v5/*:: as any*/)
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
                  (v2/*:: as any*/),
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
                        "args": null,
                        "kind": "FragmentSpread",
                        "name": "EditVCSProviderLinkFragment_link"
                      },
                      {
                        "args": null,
                        "kind": "FragmentSpread",
                        "name": "VCSProviderLinkWebhookFragment_link"
                      },
                      {
                        "args": null,
                        "kind": "FragmentSpread",
                        "name": "VCSProviderLinkDangerZoneFragment_link"
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
          (v6/*:: as any*/),
          (v7/*:: as any*/),
          (v8/*:: as any*/)
        ],
        "storageKey": null
      }
    ],
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*:: as any*/),
    "kind": "Operation",
    "name": "NewVCSProviderLinkMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*:: as any*/),
        "concreteType": "CreateWorkspaceVCSProviderLinkPayload",
        "kind": "LinkedField",
        "name": "createWorkspaceVCSProviderLink",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "WorkspaceVCSProviderLink",
            "kind": "LinkedField",
            "name": "vcsProviderLink",
            "plural": false,
            "selections": [
              (v2/*:: as any*/),
              (v3/*:: as any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "VCSProvider",
                "kind": "LinkedField",
                "name": "vcsProvider",
                "plural": false,
                "selections": [
                  (v4/*:: as any*/),
                  (v5/*:: as any*/),
                  (v2/*:: as any*/)
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
                  (v2/*:: as any*/),
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "WorkspaceVCSProviderLink",
                    "kind": "LinkedField",
                    "name": "workspaceVcsProviderLink",
                    "plural": false,
                    "selections": [
                      (v2/*:: as any*/),
                      (v3/*:: as any*/),
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
                            "name": "groupPath",
                            "storageKey": null
                          },
                          (v4/*:: as any*/),
                          (v5/*:: as any*/)
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
                  }
                ],
                "storageKey": null
              }
            ],
            "storageKey": null
          },
          (v6/*:: as any*/),
          (v7/*:: as any*/),
          (v8/*:: as any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "67d9ad681b53565ae8e0e1def3750986",
    "id": null,
    "metadata": {},
    "name": "NewVCSProviderLinkMutation",
    "operationKind": "mutation",
    "text": "mutation NewVCSProviderLinkMutation(\n  $input: CreateWorkspaceVCSProviderLinkInput!\n) {\n  createWorkspaceVCSProviderLink(input: $input) {\n    vcsProviderLink {\n      id\n      repositoryPath\n      vcsProvider {\n        type\n        autoCreateWebhooks\n        id\n      }\n      workspace {\n        id\n        workspaceVcsProviderLink {\n          id\n          ...EditVCSProviderLinkFragment_link\n          ...VCSProviderLinkWebhookFragment_link\n          ...VCSProviderLinkDangerZoneFragment_link\n        }\n      }\n    }\n    webhookToken\n    webhookUrl\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment EditVCSProviderLinkFragment_link on WorkspaceVCSProviderLink {\n  id\n  repositoryPath\n  branch\n  moduleDirectory\n  tagRegex\n  globPatterns\n  autoSpeculativePlan\n  webhookDisabled\n  vcsProvider {\n    id\n    name\n    groupPath\n    type\n    autoCreateWebhooks\n  }\n}\n\nfragment VCSProviderLinkDangerZoneFragment_link on WorkspaceVCSProviderLink {\n  id\n  repositoryPath\n  vcsProvider {\n    autoCreateWebhooks\n    id\n  }\n}\n\nfragment VCSProviderLinkWebhookFragment_link on WorkspaceVCSProviderLink {\n  id\n  repositoryPath\n  webhookDisabled\n  usesLegacyWebhookToken\n  vcsProvider {\n    type\n    autoCreateWebhooks\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "b1241fa1e6736e38a5ce1bb23df1dd63";

export default node;
