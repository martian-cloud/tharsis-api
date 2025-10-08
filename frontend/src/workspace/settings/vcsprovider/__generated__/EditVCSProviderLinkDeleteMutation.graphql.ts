/**
 * @generated SignedSource<<e8e585180a5941ff8d7cb26395be1d37>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type DeleteWorkspaceVCSProviderLinkInput = {
  clientMutationId?: string | null | undefined;
  force?: boolean | null | undefined;
  id: string;
  metadata?: ResourceMetadataInput | null | undefined;
};
export type ResourceMetadataInput = {
  version: string;
};
export type EditVCSProviderLinkDeleteMutation$variables = {
  input: DeleteWorkspaceVCSProviderLinkInput;
};
export type EditVCSProviderLinkDeleteMutation$data = {
  readonly deleteWorkspaceVCSProviderLink: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly vcsProviderLink: {
      readonly workspace: {
        readonly id: string;
        readonly workspaceVcsProviderLink: {
          readonly id: string;
        } | null | undefined;
      };
    } | null | undefined;
  };
};
export type EditVCSProviderLinkDeleteMutation = {
  response: EditVCSProviderLinkDeleteMutation$data;
  variables: EditVCSProviderLinkDeleteMutation$variables;
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
        (v2/*: any*/)
      ],
      "storageKey": null
    }
  ],
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
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "type",
      "storageKey": null
    }
  ],
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "EditVCSProviderLinkDeleteMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "DeleteWorkspaceVCSProviderLinkPayload",
        "kind": "LinkedField",
        "name": "deleteWorkspaceVCSProviderLink",
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
              (v3/*: any*/)
            ],
            "storageKey": null
          },
          (v4/*: any*/)
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
    "name": "EditVCSProviderLinkDeleteMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "DeleteWorkspaceVCSProviderLinkPayload",
        "kind": "LinkedField",
        "name": "deleteWorkspaceVCSProviderLink",
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
              (v2/*: any*/)
            ],
            "storageKey": null
          },
          (v4/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "7f2973922117b976ee4edc99111cbf77",
    "id": null,
    "metadata": {},
    "name": "EditVCSProviderLinkDeleteMutation",
    "operationKind": "mutation",
    "text": "mutation EditVCSProviderLinkDeleteMutation(\n  $input: DeleteWorkspaceVCSProviderLinkInput!\n) {\n  deleteWorkspaceVCSProviderLink(input: $input) {\n    vcsProviderLink {\n      workspace {\n        id\n        workspaceVcsProviderLink {\n          id\n        }\n      }\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "96e2730b2b426c49a8bca9dd053f7cd6";

export default node;
