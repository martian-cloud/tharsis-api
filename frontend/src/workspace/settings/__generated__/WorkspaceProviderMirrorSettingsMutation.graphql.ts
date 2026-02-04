/**
 * @generated SignedSource<<593c8f39330d455f364ddc824d7d7a35>>
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
  labels?: ReadonlyArray<WorkspaceLabelInput> | null | undefined;
  maxJobDuration?: number | null | undefined;
  metadata?: ResourceMetadataInput | null | undefined;
  preventDestroyPlan?: boolean | null | undefined;
  providerMirrorEnabled?: NamespaceProviderMirrorEnabledInput | null | undefined;
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
export type NamespaceProviderMirrorEnabledInput = {
  enabled?: boolean | null | undefined;
  inherit: boolean;
};
export type NamespaceRunnerTagsInput = {
  inherit: boolean;
  tags?: ReadonlyArray<string> | null | undefined;
};
export type WorkspaceProviderMirrorSettingsMutation$variables = {
  input: UpdateWorkspaceInput;
};
export type WorkspaceProviderMirrorSettingsMutation$data = {
  readonly updateWorkspace: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly workspace: {
      readonly providerMirrorEnabled: {
        readonly " $fragmentSpreads": FragmentRefs<"ProviderMirrorSettingsFormFragment_providerMirrorEnabled">;
      };
    } | null | undefined;
  };
};
export type WorkspaceProviderMirrorSettingsMutation = {
  response: WorkspaceProviderMirrorSettingsMutation$data;
  variables: WorkspaceProviderMirrorSettingsMutation$variables;
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
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "WorkspaceProviderMirrorSettingsMutation",
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
              {
                "alias": null,
                "args": null,
                "concreteType": "NamespaceProviderMirrorEnabled",
                "kind": "LinkedField",
                "name": "providerMirrorEnabled",
                "plural": false,
                "selections": [
                  {
                    "args": null,
                    "kind": "FragmentSpread",
                    "name": "ProviderMirrorSettingsFormFragment_providerMirrorEnabled"
                  }
                ],
                "storageKey": null
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
    "name": "WorkspaceProviderMirrorSettingsMutation",
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
              {
                "alias": null,
                "args": null,
                "concreteType": "NamespaceProviderMirrorEnabled",
                "kind": "LinkedField",
                "name": "providerMirrorEnabled",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "inherited",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "namespacePath",
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
              },
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
          (v2/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "6a3aad1e067d017980774a5410fa3003",
    "id": null,
    "metadata": {},
    "name": "WorkspaceProviderMirrorSettingsMutation",
    "operationKind": "mutation",
    "text": "mutation WorkspaceProviderMirrorSettingsMutation(\n  $input: UpdateWorkspaceInput!\n) {\n  updateWorkspace(input: $input) {\n    workspace {\n      providerMirrorEnabled {\n        ...ProviderMirrorSettingsFormFragment_providerMirrorEnabled\n      }\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment ProviderMirrorSettingsFormFragment_providerMirrorEnabled on NamespaceProviderMirrorEnabled {\n  inherited\n  namespacePath\n  value\n}\n"
  }
};
})();

(node as any).hash = "2725fe1d474832e49c2f157fec2d4056";

export default node;
