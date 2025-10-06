/**
 * @generated SignedSource<<0492719a8c926cb688088cbd28793c31>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type CreateWorkspaceInput = {
  clientMutationId?: string | null | undefined;
  description: string;
  driftDetectionEnabled?: NamespaceDriftDetectionEnabledInput | null | undefined;
  groupId?: string | null | undefined;
  groupPath?: string | null | undefined;
  maxJobDuration?: number | null | undefined;
  name: string;
  preventDestroyPlan?: boolean | null | undefined;
  runnerTags?: NamespaceRunnerTagsInput | null | undefined;
  terraformVersion?: string | null | undefined;
};
export type NamespaceDriftDetectionEnabledInput = {
  enabled?: boolean | null | undefined;
  inherit: boolean;
};
export type NamespaceRunnerTagsInput = {
  inherit: boolean;
  tags?: ReadonlyArray<string> | null | undefined;
};
export type NewWorkspaceMutation$variables = {
  connections: ReadonlyArray<string>;
  input: CreateWorkspaceInput;
};
export type NewWorkspaceMutation$data = {
  readonly createWorkspace: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly workspace: {
      readonly fullPath: string;
      readonly id: string;
      readonly name: string;
    } | null | undefined;
  };
};
export type NewWorkspaceMutation = {
  response: NewWorkspaceMutation$data;
  variables: NewWorkspaceMutation$variables;
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
  "concreteType": "Workspace",
  "kind": "LinkedField",
  "name": "workspace",
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
      "name": "name",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "fullPath",
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
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "NewWorkspaceMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "CreateWorkspacePayload",
        "kind": "LinkedField",
        "name": "createWorkspace",
        "plural": false,
        "selections": [
          (v3/*: any*/),
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
    "name": "NewWorkspaceMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "CreateWorkspacePayload",
        "kind": "LinkedField",
        "name": "createWorkspace",
        "plural": false,
        "selections": [
          (v3/*: any*/),
          {
            "alias": null,
            "args": null,
            "filters": null,
            "handle": "prependNode",
            "key": "",
            "kind": "LinkedHandle",
            "name": "workspace",
            "handleArgs": [
              {
                "kind": "Variable",
                "name": "connections",
                "variableName": "connections"
              },
              {
                "kind": "Literal",
                "name": "edgeTypeName",
                "value": "WorkspaceEdge"
              }
            ]
          },
          (v4/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "0f98c2ca6beba426776559a62ebbe56a",
    "id": null,
    "metadata": {},
    "name": "NewWorkspaceMutation",
    "operationKind": "mutation",
    "text": "mutation NewWorkspaceMutation(\n  $input: CreateWorkspaceInput!\n) {\n  createWorkspace(input: $input) {\n    workspace {\n      id\n      name\n      fullPath\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "290b3a2f2a4537b02849d811a7f7d8ae";

export default node;
