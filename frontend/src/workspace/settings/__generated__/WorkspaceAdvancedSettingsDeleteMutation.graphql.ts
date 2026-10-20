/**
 * @generated SignedSource<<180e75bf74ad73e2bd482bb543a22258>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type DeleteWorkspaceInput = {
  clientMutationId?: string | null | undefined;
  force?: boolean | null | undefined;
  id?: string | null | undefined;
  metadata?: ResourceMetadataInput | null | undefined;
  workspacePath?: string | null | undefined;
};
export type ResourceMetadataInput = {
  version: string;
};
export type WorkspaceAdvancedSettingsDeleteMutation$variables = {
  input: DeleteWorkspaceInput;
};
export type WorkspaceAdvancedSettingsDeleteMutation$data = {
  readonly deleteWorkspace: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type WorkspaceAdvancedSettingsDeleteMutation = {
  response: WorkspaceAdvancedSettingsDeleteMutation$data;
  variables: WorkspaceAdvancedSettingsDeleteMutation$variables;
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
    "concreteType": "DeleteWorkspacePayload",
    "kind": "LinkedField",
    "name": "deleteWorkspace",
    "plural": false,
    "selections": [
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
    "name": "WorkspaceAdvancedSettingsDeleteMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "WorkspaceAdvancedSettingsDeleteMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "d20dc57ffe13f00e4e170fcdbb3564ac",
    "id": null,
    "metadata": {},
    "name": "WorkspaceAdvancedSettingsDeleteMutation",
    "operationKind": "mutation",
    "text": "mutation WorkspaceAdvancedSettingsDeleteMutation(\n  $input: DeleteWorkspaceInput!\n) {\n  deleteWorkspace(input: $input) {\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "a9c90866ec533812225e51ccd893359d";

export default node;
