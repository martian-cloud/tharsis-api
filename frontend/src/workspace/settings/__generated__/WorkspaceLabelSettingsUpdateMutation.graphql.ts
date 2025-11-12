/**
 * @generated SignedSource<<3a146984aa03c329b80ea11de359ad55>>
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
  labels?: ReadonlyArray<WorkspaceLabelInput> | null | undefined;
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
export type WorkspaceLabelInput = {
  key: string;
  value: string;
};
export type ResourceMetadataInput = {
  version: string;
};
export type NamespaceRunnerTagsInput = {
  inherit: boolean;
  tags?: ReadonlyArray<string> | null | undefined;
};
export type WorkspaceLabelSettingsUpdateMutation$variables = {
  input: UpdateWorkspaceInput;
};
export type WorkspaceLabelSettingsUpdateMutation$data = {
  readonly updateWorkspace: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly workspace: {
      readonly fullPath: string;
      readonly id: string;
      readonly labels: ReadonlyArray<{
        readonly key: string;
        readonly value: string;
      }>;
    } | null | undefined;
  };
};
export type WorkspaceLabelSettingsUpdateMutation = {
  response: WorkspaceLabelSettingsUpdateMutation$data;
  variables: WorkspaceLabelSettingsUpdateMutation$variables;
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
            "name": "fullPath",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "concreteType": "WorkspaceLabel",
            "kind": "LinkedField",
            "name": "labels",
            "plural": true,
            "selections": [
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "key",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "value",
                "storageKey": null
              }
            ],
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
    "name": "WorkspaceLabelSettingsUpdateMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "WorkspaceLabelSettingsUpdateMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "374fcc93b981bdfbc560763277e2fe27",
    "id": null,
    "metadata": {},
    "name": "WorkspaceLabelSettingsUpdateMutation",
    "operationKind": "mutation",
    "text": "mutation WorkspaceLabelSettingsUpdateMutation(\n  $input: UpdateWorkspaceInput!\n) {\n  updateWorkspace(input: $input) {\n    workspace {\n      id\n      fullPath\n      labels {\n        key\n        value\n      }\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "daef5d91d32ae74315b63dd6dbc7e167";

export default node;
