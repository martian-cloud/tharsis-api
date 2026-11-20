/**
 * @generated SignedSource<<1d5af25656fb99a4d6941c3765ce004e>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type UpdateWorkspaceVCSProviderLinkInput = {
  autoSpeculativePlan?: boolean | null | undefined;
  branch?: string | null | undefined;
  clientMutationId?: string | null | undefined;
  globPatterns: ReadonlyArray<string>;
  id: string;
  metadata?: ResourceMetadataInput | null | undefined;
  moduleDirectory?: string | null | undefined;
  tagRegex?: string | null | undefined;
  webhookDisabled?: boolean | null | undefined;
};
export type ResourceMetadataInput = {
  version: string;
};
export type EditVCSProviderLinkMutation$variables = {
  input: UpdateWorkspaceVCSProviderLinkInput;
};
export type EditVCSProviderLinkMutation$data = {
  readonly updateWorkspaceVCSProviderLink: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly vcsProviderLink: {
      readonly id: string;
      readonly " $fragmentSpreads": FragmentRefs<"EditVCSProviderLinkFragment_link" | "VCSProviderLinkWebhookFragment_link">;
    } | null | undefined;
  };
};
export type EditVCSProviderLinkMutation = {
  response: EditVCSProviderLinkMutation$data;
  variables: EditVCSProviderLinkMutation$variables;
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
  "name": "type",
  "storageKey": null
},
v4 = {
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
    (v3/*:: as any*/)
  ],
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*:: as any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "EditVCSProviderLinkMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*:: as any*/),
        "concreteType": "UpdateWorkspaceVCSProviderLinkPayload",
        "kind": "LinkedField",
        "name": "updateWorkspaceVCSProviderLink",
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
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "EditVCSProviderLinkFragment_link"
              },
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "VCSProviderLinkWebhookFragment_link"
              }
            ],
            "storageKey": null
          },
          (v4/*:: as any*/)
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
    "name": "EditVCSProviderLinkMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*:: as any*/),
        "concreteType": "UpdateWorkspaceVCSProviderLinkPayload",
        "kind": "LinkedField",
        "name": "updateWorkspaceVCSProviderLink",
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
                  (v3/*:: as any*/),
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
          (v4/*:: as any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "3056b86c3450a3184f5f0157e3d70dd9",
    "id": null,
    "metadata": {},
    "name": "EditVCSProviderLinkMutation",
    "operationKind": "mutation",
    "text": "mutation EditVCSProviderLinkMutation(\n  $input: UpdateWorkspaceVCSProviderLinkInput!\n) {\n  updateWorkspaceVCSProviderLink(input: $input) {\n    vcsProviderLink {\n      id\n      ...EditVCSProviderLinkFragment_link\n      ...VCSProviderLinkWebhookFragment_link\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment EditVCSProviderLinkFragment_link on WorkspaceVCSProviderLink {\n  id\n  repositoryPath\n  branch\n  moduleDirectory\n  tagRegex\n  globPatterns\n  autoSpeculativePlan\n  webhookDisabled\n  vcsProvider {\n    id\n    name\n    groupPath\n    type\n    autoCreateWebhooks\n  }\n}\n\nfragment VCSProviderLinkWebhookFragment_link on WorkspaceVCSProviderLink {\n  id\n  repositoryPath\n  webhookDisabled\n  usesLegacyWebhookToken\n  vcsProvider {\n    type\n    autoCreateWebhooks\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "0f55bfbaf07e0be094cf65f85e4991e0";

export default node;
