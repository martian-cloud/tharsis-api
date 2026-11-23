/**
 * @generated SignedSource<<028c6690d24fd532dcfebc8ab4d59375>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type ResetWorkspaceVCSProviderLinkWebhookTokenInput = {
  clientMutationId?: string | null | undefined;
  id: string;
};
export type VCSProviderLinkWebhookResetMutation$variables = {
  input: ResetWorkspaceVCSProviderLinkWebhookTokenInput;
};
export type VCSProviderLinkWebhookResetMutation$data = {
  readonly resetWorkspaceVCSProviderLinkWebhookToken: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly vcsProviderLink: {
      readonly id: string;
      readonly usesLegacyWebhookToken: boolean;
    } | null | undefined;
    readonly webhookToken: string | null | undefined;
    readonly webhookUrl: string | null | undefined;
  };
};
export type VCSProviderLinkWebhookResetMutation = {
  response: VCSProviderLinkWebhookResetMutation$data;
  variables: VCSProviderLinkWebhookResetMutation$variables;
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
    "alias": null,
    "args": [
      {
        "kind": "Variable",
        "name": "input",
        "variableName": "input"
      }
    ],
    "concreteType": "ResetWorkspaceVCSProviderLinkWebhookTokenPayload",
    "kind": "LinkedField",
    "name": "resetWorkspaceVCSProviderLinkWebhookToken",
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
            "kind": "ScalarField",
            "name": "id",
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
        "args": null,
        "kind": "ScalarField",
        "name": "webhookToken",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "webhookUrl",
        "storageKey": null
      },
      {
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
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "type",
            "storageKey": null
          }
        ],
        "storageKey": null
      }
    ],
    "storageKey": null
  }
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*:: as any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "VCSProviderLinkWebhookResetMutation",
    "selections": (v1/*:: as any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*:: as any*/),
    "kind": "Operation",
    "name": "VCSProviderLinkWebhookResetMutation",
    "selections": (v1/*:: as any*/)
  },
  "params": {
    "cacheID": "7d6d1f9d170641b1b661470a6c8c161c",
    "id": null,
    "metadata": {},
    "name": "VCSProviderLinkWebhookResetMutation",
    "operationKind": "mutation",
    "text": "mutation VCSProviderLinkWebhookResetMutation(\n  $input: ResetWorkspaceVCSProviderLinkWebhookTokenInput!\n) {\n  resetWorkspaceVCSProviderLinkWebhookToken(input: $input) {\n    vcsProviderLink {\n      id\n      usesLegacyWebhookToken\n    }\n    webhookToken\n    webhookUrl\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "8354f23eb12670b60759acc82f04503d";

export default node;
