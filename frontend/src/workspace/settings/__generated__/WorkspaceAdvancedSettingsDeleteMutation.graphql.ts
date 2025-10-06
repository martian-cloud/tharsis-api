/**
 * @generated SignedSource<<932fe9126141c5c39440e06c761d5463>>
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
  connections: ReadonlyArray<string>;
  input: DeleteWorkspaceInput;
};
export type WorkspaceAdvancedSettingsDeleteMutation$data = {
  readonly deleteWorkspace: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly workspace: {
      readonly id: string;
    } | null | undefined;
  };
};
export type WorkspaceAdvancedSettingsDeleteMutation = {
  response: WorkspaceAdvancedSettingsDeleteMutation$data;
  variables: WorkspaceAdvancedSettingsDeleteMutation$variables;
};

const node: ConcreteRequest = (function(){
var v0 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "connections"
},
v1 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "input"
},
v2 = [
  {
    "kind": "Variable",
    "name": "input",
    "variableName": "input"
  }
],
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
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
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "WorkspaceAdvancedSettingsDeleteMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "DeleteWorkspacePayload",
        "kind": "LinkedField",
        "name": "deleteWorkspace",
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
    "argumentDefinitions": [
      (v1/*: any*/),
      (v0/*: any*/)
    ],
    "kind": "Operation",
    "name": "WorkspaceAdvancedSettingsDeleteMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "DeleteWorkspacePayload",
        "kind": "LinkedField",
        "name": "deleteWorkspace",
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
              (v3/*: any*/),
              {
                "alias": null,
                "args": null,
                "filters": null,
                "handle": "deleteEdge",
                "key": "",
                "kind": "ScalarHandle",
                "name": "id",
                "handleArgs": [
                  {
                    "kind": "Variable",
                    "name": "connections",
                    "variableName": "connections"
                  }
                ]
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
    "cacheID": "716f6336d5d3076e46fc93710feb00c2",
    "id": null,
    "metadata": {},
    "name": "WorkspaceAdvancedSettingsDeleteMutation",
    "operationKind": "mutation",
    "text": "mutation WorkspaceAdvancedSettingsDeleteMutation(\n  $input: DeleteWorkspaceInput!\n) {\n  deleteWorkspace(input: $input) {\n    workspace {\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "c6ed157ff307ee2477d144ab9a986fa0";

export default node;
