/**
 * @generated SignedSource<<e390f4399d2c67ca4761d56000c483ad>>
 * @lightSyntaxTransform
 * @nogrep
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
      readonly vcsProvider: {
        readonly autoCreateWebhooks: boolean;
        readonly type: VCSProviderType;
      };
      readonly workspace: {
        readonly id: string;
        readonly workspaceVcsProviderLink: {
          readonly autoSpeculativePlan: boolean;
          readonly branch: string;
          readonly createdBy: string;
          readonly createdByActor: {
            readonly " $fragmentSpreads": FragmentRefs<"ActorDisplayFragment_actor">;
          } | null | undefined;
          readonly globPatterns: ReadonlyArray<string>;
          readonly id: string;
          readonly metadata: {
            readonly createdAt: any;
          };
          readonly moduleDirectory: string | null | undefined;
          readonly repositoryPath: string;
          readonly tagRegex: string | null | undefined;
          readonly webhookDisabled: boolean;
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
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "createdBy",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "repositoryPath",
  "storageKey": null
},
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "branch",
  "storageKey": null
},
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "moduleDirectory",
  "storageKey": null
},
v8 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "tagRegex",
  "storageKey": null
},
v9 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "globPatterns",
  "storageKey": null
},
v10 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "autoSpeculativePlan",
  "storageKey": null
},
v11 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "webhookDisabled",
  "storageKey": null
},
v12 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "type",
  "storageKey": null
},
v13 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "autoCreateWebhooks",
  "storageKey": null
},
v14 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "webhookToken",
  "storageKey": null
},
v15 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "webhookUrl",
  "storageKey": null
},
v16 = {
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
    (v12/*: any*/)
  ],
  "storageKey": null
},
v17 = [
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
    "name": "NewVCSProviderLinkMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
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
              {
                "alias": null,
                "args": null,
                "concreteType": "Workspace",
                "kind": "LinkedField",
                "name": "workspace",
                "plural": false,
                "selections": [
                  (v2/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "WorkspaceVCSProviderLink",
                    "kind": "LinkedField",
                    "name": "workspaceVcsProviderLink",
                    "plural": false,
                    "selections": [
                      (v2/*: any*/),
                      (v3/*: any*/),
                      (v4/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": null,
                        "kind": "LinkedField",
                        "name": "createdByActor",
                        "plural": false,
                        "selections": [
                          {
                            "args": null,
                            "kind": "FragmentSpread",
                            "name": "ActorDisplayFragment_actor"
                          }
                        ],
                        "storageKey": null
                      },
                      (v5/*: any*/),
                      (v6/*: any*/),
                      (v7/*: any*/),
                      (v8/*: any*/),
                      (v9/*: any*/),
                      (v10/*: any*/),
                      (v11/*: any*/)
                    ],
                    "storageKey": null
                  }
                ],
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
                  (v12/*: any*/),
                  (v13/*: any*/)
                ],
                "storageKey": null
              }
            ],
            "storageKey": null
          },
          (v14/*: any*/),
          (v15/*: any*/),
          (v16/*: any*/)
        ],
        "storageKey": null
      }
    ],
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "NewVCSProviderLinkMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
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
              {
                "alias": null,
                "args": null,
                "concreteType": "Workspace",
                "kind": "LinkedField",
                "name": "workspace",
                "plural": false,
                "selections": [
                  (v2/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "WorkspaceVCSProviderLink",
                    "kind": "LinkedField",
                    "name": "workspaceVcsProviderLink",
                    "plural": false,
                    "selections": [
                      (v2/*: any*/),
                      (v3/*: any*/),
                      (v4/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": null,
                        "kind": "LinkedField",
                        "name": "createdByActor",
                        "plural": false,
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "__typename",
                            "storageKey": null
                          },
                          {
                            "kind": "TypeDiscriminator",
                            "abstractKey": "__isActor"
                          },
                          (v12/*: any*/),
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
                                  (v2/*: any*/)
                                ],
                                "storageKey": null
                              }
                            ],
                            "type": "UserActor",
                            "abstractKey": null
                          },
                          {
                            "kind": "InlineFragment",
                            "selections": (v17/*: any*/),
                            "type": "ServiceAccountActor",
                            "abstractKey": null
                          },
                          {
                            "kind": "InlineFragment",
                            "selections": (v17/*: any*/),
                            "type": "VCSProviderActor",
                            "abstractKey": null
                          },
                          {
                            "kind": "InlineFragment",
                            "selections": (v17/*: any*/),
                            "type": "JobActor",
                            "abstractKey": null
                          }
                        ],
                        "storageKey": null
                      },
                      (v5/*: any*/),
                      (v6/*: any*/),
                      (v7/*: any*/),
                      (v8/*: any*/),
                      (v9/*: any*/),
                      (v10/*: any*/),
                      (v11/*: any*/)
                    ],
                    "storageKey": null
                  }
                ],
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
                  (v12/*: any*/),
                  (v13/*: any*/),
                  (v2/*: any*/)
                ],
                "storageKey": null
              },
              (v2/*: any*/)
            ],
            "storageKey": null
          },
          (v14/*: any*/),
          (v15/*: any*/),
          (v16/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "b6208bdefd11abf2616ca6f85c1b332d",
    "id": null,
    "metadata": {},
    "name": "NewVCSProviderLinkMutation",
    "operationKind": "mutation",
    "text": "mutation NewVCSProviderLinkMutation(\n  $input: CreateWorkspaceVCSProviderLinkInput!\n) {\n  createWorkspaceVCSProviderLink(input: $input) {\n    vcsProviderLink {\n      workspace {\n        id\n        workspaceVcsProviderLink {\n          id\n          metadata {\n            createdAt\n          }\n          createdBy\n          createdByActor {\n            __typename\n            ...ActorDisplayFragment_actor\n          }\n          repositoryPath\n          branch\n          moduleDirectory\n          tagRegex\n          globPatterns\n          autoSpeculativePlan\n          webhookDisabled\n        }\n      }\n      vcsProvider {\n        type\n        autoCreateWebhooks\n        id\n      }\n      id\n    }\n    webhookToken\n    webhookUrl\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment ActorDisplayFragment_actor on Actor {\n  __isActor: __typename\n  type\n  principalId\n  ... on UserActor {\n    user {\n      email\n      id\n    }\n  }\n  ... on ServiceAccountActor {\n    trnSnapshot\n  }\n  ... on VCSProviderActor {\n    trnSnapshot\n  }\n  ... on JobActor {\n    trnSnapshot\n  }\n}\n"
  }
};
})();

(node as any).hash = "cd1cc9e174e183c66f3b66f5234fa7eb";

export default node;
