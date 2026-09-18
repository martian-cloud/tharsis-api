/**
 * @generated SignedSource<<776078348298e4e2b821d3a816e4194b>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type SetWorkspaceRoleBindingInput = {
  clientMutationId?: string | null | undefined;
  roleId?: string | null | undefined;
  workspaceId: string;
};
export type WorkspaceRoleBindingSetMutation$variables = {
  input: SetWorkspaceRoleBindingInput;
};
export type WorkspaceRoleBindingSetMutation$data = {
  readonly setWorkspaceRoleBinding: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly workspace: {
      readonly " $fragmentSpreads": FragmentRefs<"WorkspaceRoleBindingRoleFragment_workspace">;
    } | null | undefined;
  };
};
export type WorkspaceRoleBindingSetMutation = {
  response: WorkspaceRoleBindingSetMutation$data;
  variables: WorkspaceRoleBindingSetMutation$variables;
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
},
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "WorkspaceRoleBindingSetMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "SetWorkspaceRoleBindingPayload",
        "kind": "LinkedField",
        "name": "setWorkspaceRoleBinding",
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
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "WorkspaceRoleBindingRoleFragment_workspace"
              }
            ],
            "storageKey": null
          },
          (v2/*: any*/)
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
    "name": "WorkspaceRoleBindingSetMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "SetWorkspaceRoleBindingPayload",
        "kind": "LinkedField",
        "name": "setWorkspaceRoleBinding",
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
              {
                "alias": null,
                "args": null,
                "concreteType": "WorkspaceRoleBinding",
                "kind": "LinkedField",
                "name": "roleBinding",
                "plural": false,
                "selections": [
                  (v3/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "Role",
                    "kind": "LinkedField",
                    "name": "role",
                    "plural": false,
                    "selections": [
                      (v3/*: any*/),
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
                        "name": "description",
                        "storageKey": null
                      }
                    ],
                    "storageKey": null
                  }
                ],
                "storageKey": null
              },
              (v3/*: any*/)
            ],
            "storageKey": null
          },
          (v2/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "4be557fe8ebf87c77a4dc1d97b68e668",
    "id": null,
    "metadata": {},
    "name": "WorkspaceRoleBindingSetMutation",
    "operationKind": "mutation",
    "text": "mutation WorkspaceRoleBindingSetMutation(\n  $input: SetWorkspaceRoleBindingInput!\n) {\n  setWorkspaceRoleBinding(input: $input) {\n    workspace {\n      ...WorkspaceRoleBindingRoleFragment_workspace\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment WorkspaceRoleBindingRoleFragment_workspace on Workspace {\n  roleBinding {\n    id\n    role {\n      id\n      name\n      description\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "4f47ecd78b9b8e42b74517b4aef735d5";

export default node;
