/**
 * @generated SignedSource<<16a023de513652fb6f77fa344e008a51>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
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
export type WorkspaceRunnerSettingsMutation$variables = {
  input: UpdateWorkspaceInput;
};
export type WorkspaceRunnerSettingsMutation$data = {
  readonly updateWorkspace: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly workspace: {
      readonly id: string;
      readonly runnerTags: {
        readonly inherited: boolean;
        readonly value: ReadonlyArray<string>;
      };
      readonly " $fragmentSpreads": FragmentRefs<"WorkspaceRunnerSettingsFragment_workspace">;
    } | null | undefined;
  };
};
export type WorkspaceRunnerSettingsMutation = {
  response: WorkspaceRunnerSettingsMutation$data;
  variables: WorkspaceRunnerSettingsMutation$variables;
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
  "kind": "ScalarField",
  "name": "inherited",
  "storageKey": null
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "value",
  "storageKey": null
},
v5 = {
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
    "name": "WorkspaceRunnerSettingsMutation",
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
              {
                "alias": null,
                "args": null,
                "concreteType": "NamespaceRunnerTags",
                "kind": "LinkedField",
                "name": "runnerTags",
                "plural": false,
                "selections": [
                  (v3/*: any*/),
                  (v4/*: any*/)
                ],
                "storageKey": null
              },
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "WorkspaceRunnerSettingsFragment_workspace"
              }
            ],
            "storageKey": null
          },
          (v5/*: any*/)
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
    "name": "WorkspaceRunnerSettingsMutation",
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
              {
                "alias": null,
                "args": null,
                "concreteType": "NamespaceRunnerTags",
                "kind": "LinkedField",
                "name": "runnerTags",
                "plural": false,
                "selections": [
                  (v3/*: any*/),
                  (v4/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "namespacePath",
                    "storageKey": null
                  }
                ],
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
          (v5/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "39f90fbe1bf29f85241d2a80010969ce",
    "id": null,
    "metadata": {},
    "name": "WorkspaceRunnerSettingsMutation",
    "operationKind": "mutation",
    "text": "mutation WorkspaceRunnerSettingsMutation(\n  $input: UpdateWorkspaceInput!\n) {\n  updateWorkspace(input: $input) {\n    workspace {\n      id\n      runnerTags {\n        inherited\n        value\n      }\n      ...WorkspaceRunnerSettingsFragment_workspace\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment RunnerSettingsForm_runnerTags on NamespaceRunnerTags {\n  inherited\n  namespacePath\n  value\n}\n\nfragment WorkspaceRunnerSettingsFragment_workspace on Workspace {\n  fullPath\n  runnerTags {\n    inherited\n    namespacePath\n    value\n    ...RunnerSettingsForm_runnerTags\n  }\n}\n"
  }
};
})();

(node as any).hash = "f8523994e72ad37ae195c6ef8e0101c8";

export default node;
