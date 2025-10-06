/**
 * @generated SignedSource<<ca83413641f690af6be39e19978050ae>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
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
        }
      ],
      "storageKey": null
    }
  ],
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
    (v4/*: any*/)
  ],
  "storageKey": null
};
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
              (v3/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "VCSProvider",
                "kind": "LinkedField",
                "name": "vcsProvider",
                "plural": false,
                "selections": [
                  (v4/*: any*/),
                  (v5/*: any*/)
                ],
                "storageKey": null
              }
            ],
            "storageKey": null
          },
          (v6/*: any*/),
          (v7/*: any*/),
          (v8/*: any*/)
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
              (v3/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "VCSProvider",
                "kind": "LinkedField",
                "name": "vcsProvider",
                "plural": false,
                "selections": [
                  (v4/*: any*/),
                  (v5/*: any*/),
                  (v2/*: any*/)
                ],
                "storageKey": null
              },
              (v2/*: any*/)
            ],
            "storageKey": null
          },
          (v6/*: any*/),
          (v7/*: any*/),
          (v8/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "8a2f52f01c59ff0eaf396b03e2e66c83",
    "id": null,
    "metadata": {},
    "name": "NewVCSProviderLinkMutation",
    "operationKind": "mutation",
    "text": "mutation NewVCSProviderLinkMutation(\n  $input: CreateWorkspaceVCSProviderLinkInput!\n) {\n  createWorkspaceVCSProviderLink(input: $input) {\n    vcsProviderLink {\n      workspace {\n        id\n        workspaceVcsProviderLink {\n          id\n          metadata {\n            createdAt\n          }\n          createdBy\n          repositoryPath\n          branch\n          moduleDirectory\n          tagRegex\n          globPatterns\n          autoSpeculativePlan\n          webhookDisabled\n        }\n      }\n      vcsProvider {\n        type\n        autoCreateWebhooks\n        id\n      }\n      id\n    }\n    webhookToken\n    webhookUrl\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "e5e6a4e9c0b6c40421af1d4fd294ae55";

export default node;
