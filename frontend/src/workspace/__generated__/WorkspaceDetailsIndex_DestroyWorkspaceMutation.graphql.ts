/**
 * @generated SignedSource<<b015127f8df8332bfcedee2c94c9da7c>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type DestroyWorkspaceInput = {
  clientMutationId?: string | null | undefined;
  workspaceId?: string | null | undefined;
  workspacePath?: string | null | undefined;
};
export type WorkspaceDetailsIndex_DestroyWorkspaceMutation$variables = {
  input: DestroyWorkspaceInput;
};
export type WorkspaceDetailsIndex_DestroyWorkspaceMutation$data = {
  readonly destroyWorkspace: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly run: {
      readonly id: string;
    } | null | undefined;
  };
};
export type WorkspaceDetailsIndex_DestroyWorkspaceMutation = {
  response: WorkspaceDetailsIndex_DestroyWorkspaceMutation$data;
  variables: WorkspaceDetailsIndex_DestroyWorkspaceMutation$variables;
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
    "concreteType": "DestroyWorkspacePayload",
    "kind": "LinkedField",
    "name": "destroyWorkspace",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "Run",
        "kind": "LinkedField",
        "name": "run",
        "plural": false,
        "selections": [
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
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "WorkspaceDetailsIndex_DestroyWorkspaceMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "WorkspaceDetailsIndex_DestroyWorkspaceMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "9aef538f33826cc68265b9c20f5f5217",
    "id": null,
    "metadata": {},
    "name": "WorkspaceDetailsIndex_DestroyWorkspaceMutation",
    "operationKind": "mutation",
    "text": "mutation WorkspaceDetailsIndex_DestroyWorkspaceMutation(\n  $input: DestroyWorkspaceInput!\n) {\n  destroyWorkspace(input: $input) {\n    run {\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "94d84f0dd87a4f204847996ffc92fa71";

export default node;
