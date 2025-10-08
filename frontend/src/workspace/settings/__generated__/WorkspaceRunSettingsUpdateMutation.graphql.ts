/**
 * @generated SignedSource<<c116011cec1e083e2699e71855a7bc1c>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type UpdateWorkspaceInput = {
  clientMutationId?: string | null | undefined;
  description?: string | null | undefined;
  driftDetectionEnabled?: NamespaceDriftDetectionEnabledInput | null | undefined;
  id?: string | null | undefined;
  maxJobDuration?: number | null | undefined;
  metadata?: ResourceMetadataInput | null | undefined;
  preventDestroyPlan?: boolean | null | undefined;
  runnerTags?: NamespaceRunnerTagsInput | null | undefined;
  terraformVersion?: string | null | undefined;
  workspacePath?: string | null | undefined;
};
export type NamespaceDriftDetectionEnabledInput = {
  enabled?: boolean | null | undefined;
  inherit: boolean;
};
export type ResourceMetadataInput = {
  version: string;
};
export type NamespaceRunnerTagsInput = {
  inherit: boolean;
  tags?: ReadonlyArray<string> | null | undefined;
};
export type WorkspaceRunSettingsUpdateMutation$variables = {
  input: UpdateWorkspaceInput;
};
export type WorkspaceRunSettingsUpdateMutation$data = {
  readonly updateWorkspace: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly workspace: {
      readonly fullPath: string;
      readonly maxJobDuration: number;
      readonly preventDestroyPlan: boolean;
      readonly terraformVersion: string;
    } | null | undefined;
  };
};
export type WorkspaceRunSettingsUpdateMutation = {
  response: WorkspaceRunSettingsUpdateMutation$data;
  variables: WorkspaceRunSettingsUpdateMutation$variables;
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
  "name": "maxJobDuration",
  "storageKey": null
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "terraformVersion",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "preventDestroyPlan",
  "storageKey": null
},
v6 = {
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
    "name": "WorkspaceRunSettingsUpdateMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "UpdateWorkspacePayload",
        "kind": "LinkedField",
        "name": "updateWorkspace",
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
              (v4/*: any*/),
              (v5/*: any*/)
            ],
            "storageKey": null
          },
          (v6/*: any*/)
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
    "name": "WorkspaceRunSettingsUpdateMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "UpdateWorkspacePayload",
        "kind": "LinkedField",
        "name": "updateWorkspace",
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
              (v4/*: any*/),
              (v5/*: any*/),
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
          (v6/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "eb8b32157a4e82bb2c8963ec1c888cff",
    "id": null,
    "metadata": {},
    "name": "WorkspaceRunSettingsUpdateMutation",
    "operationKind": "mutation",
    "text": "mutation WorkspaceRunSettingsUpdateMutation(\n  $input: UpdateWorkspaceInput!\n) {\n  updateWorkspace(input: $input) {\n    workspace {\n      fullPath\n      maxJobDuration\n      terraformVersion\n      preventDestroyPlan\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "cc707e1576fb4c638158da024851a016";

export default node;
