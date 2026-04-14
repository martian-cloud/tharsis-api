/**
 * @generated SignedSource<<7519c6853d4ee50c9f89f28e0d9f7dec>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type ReconcileWorkspaceInput = {
  clientMutationId?: string | null | undefined;
  workspaceId: string;
};
export type WorkspaceDetailsIndex_ReconcileWorkspaceMutation$variables = {
  input: ReconcileWorkspaceInput;
};
export type WorkspaceDetailsIndex_ReconcileWorkspaceMutation$data = {
  readonly reconcileWorkspace: {
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
export type WorkspaceDetailsIndex_ReconcileWorkspaceMutation = {
  response: WorkspaceDetailsIndex_ReconcileWorkspaceMutation$data;
  variables: WorkspaceDetailsIndex_ReconcileWorkspaceMutation$variables;
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
    "concreteType": "ReconcileWorkspacePayload",
    "kind": "LinkedField",
    "name": "reconcileWorkspace",
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
    "name": "WorkspaceDetailsIndex_ReconcileWorkspaceMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "WorkspaceDetailsIndex_ReconcileWorkspaceMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "ad3fcf07943c6fc95fd04155ef79722c",
    "id": null,
    "metadata": {},
    "name": "WorkspaceDetailsIndex_ReconcileWorkspaceMutation",
    "operationKind": "mutation",
    "text": "mutation WorkspaceDetailsIndex_ReconcileWorkspaceMutation(\n  $input: ReconcileWorkspaceInput!\n) {\n  reconcileWorkspace(input: $input) {\n    run {\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "5616ab351abe63948ea2c9562410216e";

export default node;
