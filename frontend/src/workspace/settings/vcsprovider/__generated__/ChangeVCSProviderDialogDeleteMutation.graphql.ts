/**
 * @generated SignedSource<<359b88ea09ac88cf00165dee599ff26d>>
 * @lightSyntaxTransform
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
export type ChangeVCSProviderDialogDeleteMutation$variables = {
  input: DeleteWorkspaceVCSProviderLinkInput;
};
export type ChangeVCSProviderDialogDeleteMutation$data = {
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
export type ChangeVCSProviderDialogDeleteMutation = {
  response: ChangeVCSProviderDialogDeleteMutation$data;
  variables: ChangeVCSProviderDialogDeleteMutation$variables;
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
    (v2/*:: as any*/),
    {
      "alias": null,
      "args": null,
      "concreteType": "WorkspaceVCSProviderLink",
      "kind": "LinkedField",
      "name": "workspaceVcsProviderLink",
      "plural": false,
      "selections": [
        (v2/*:: as any*/)
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
    "argumentDefinitions": (v0/*:: as any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "ChangeVCSProviderDialogDeleteMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*:: as any*/),
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
              (v3/*:: as any*/)
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
    "name": "ChangeVCSProviderDialogDeleteMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*:: as any*/),
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
              (v3/*:: as any*/),
              (v2/*:: as any*/)
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
    "cacheID": "c7da926f1fc924529626a55d17e6726a",
    "id": null,
    "metadata": {},
    "name": "ChangeVCSProviderDialogDeleteMutation",
    "operationKind": "mutation",
    "text": "mutation ChangeVCSProviderDialogDeleteMutation(\n  $input: DeleteWorkspaceVCSProviderLinkInput!\n) {\n  deleteWorkspaceVCSProviderLink(input: $input) {\n    vcsProviderLink {\n      workspace {\n        id\n        workspaceVcsProviderLink {\n          id\n        }\n      }\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "890b4d763d910c454fd2c480d2cd4a35";

export default node;
