/**
 * @generated SignedSource<<7369d43dbcb184cccf22166a56d20629>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type LockWorkspaceInput = {
  clientMutationId?: string | null | undefined;
  workspaceId?: string | null | undefined;
  workspacePath?: string | null | undefined;
};
export type WorkspaceStateSettingsLockWorkspaceMutation$variables = {
  input: LockWorkspaceInput;
};
export type WorkspaceStateSettingsLockWorkspaceMutation$data = {
  readonly lockWorkspace: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly workspace: {
      readonly fullPath: string;
      readonly locked: boolean;
    } | null | undefined;
  };
};
export type WorkspaceStateSettingsLockWorkspaceMutation = {
  response: WorkspaceStateSettingsLockWorkspaceMutation$data;
  variables: WorkspaceStateSettingsLockWorkspaceMutation$variables;
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
  "name": "fullPath",
  "storageKey": null
},
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "locked",
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
    "name": "WorkspaceStateSettingsLockWorkspaceMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "LockWorkspacePayload",
        "kind": "LinkedField",
        "name": "lockWorkspace",
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
    "name": "WorkspaceStateSettingsLockWorkspaceMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "LockWorkspacePayload",
        "kind": "LinkedField",
        "name": "lockWorkspace",
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
              (v3/*: any*/),
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "id",
                "storageKey": null
              }
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
    "cacheID": "0fdc8a5fba8d222306528fd5c22c92b8",
    "id": null,
    "metadata": {},
    "name": "WorkspaceStateSettingsLockWorkspaceMutation",
    "operationKind": "mutation",
    "text": "mutation WorkspaceStateSettingsLockWorkspaceMutation(\n  $input: LockWorkspaceInput!\n) {\n  lockWorkspace(input: $input) {\n    workspace {\n      fullPath\n      locked\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "75ab4c200195e2f5c608702763fbece2";

export default node;
