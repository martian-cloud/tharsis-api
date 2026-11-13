/**
 * @generated SignedSource<<182228a7575e97dc289511ec69622f86>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type NamespaceOutputVisibilityLevel = "block_access" | "direct_group_and_subgroups" | "direct_group_only" | "global" | "root_group" | "%future added value";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type UpdateGroupInput = {
  clientMutationId?: string | null | undefined;
  description?: string | null | undefined;
  driftDetectionEnabled?: NamespaceDriftDetectionEnabledInput | null | undefined;
  groupPath?: string | null | undefined;
  id?: string | null | undefined;
  metadata?: ResourceMetadataInput | null | undefined;
  outputVisibility?: NamespaceOutputVisibilityInput | null | undefined;
  providerMirrorEnabled?: NamespaceProviderMirrorEnabledInput | null | undefined;
  runnerTags?: NamespaceRunnerTagsInput | null | undefined;
};
export type NamespaceDriftDetectionEnabledInput = {
  enabled?: boolean | null | undefined;
  inherit: boolean;
};
export type ResourceMetadataInput = {
  version: string;
};
export type NamespaceOutputVisibilityInput = {
  inherit: boolean;
  visibility?: NamespaceOutputVisibilityLevel | null | undefined;
};
export type NamespaceProviderMirrorEnabledInput = {
  enabled?: boolean | null | undefined;
  inherit: boolean;
};
export type NamespaceRunnerTagsInput = {
  inherit: boolean;
  tags?: ReadonlyArray<string> | null | undefined;
};
export type GroupOutputVisibilitySettingsMutation$variables = {
  input: UpdateGroupInput;
};
export type GroupOutputVisibilitySettingsMutation$data = {
  readonly updateGroup: {
    readonly group: {
      readonly outputVisibility: {
        readonly " $fragmentSpreads": FragmentRefs<"OutputVisibilitySettingsFormFragment_outputVisibility">;
      };
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type GroupOutputVisibilitySettingsMutation = {
  response: GroupOutputVisibilitySettingsMutation$data;
  variables: GroupOutputVisibilitySettingsMutation$variables;
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
    "argumentDefinitions": (v0/*:: as any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "GroupOutputVisibilitySettingsMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*:: as any*/),
        "concreteType": "UpdateGroupPayload",
        "kind": "LinkedField",
        "name": "updateGroup",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "Group",
            "kind": "LinkedField",
            "name": "group",
            "plural": false,
            "selections": [
              {
                "alias": null,
                "args": null,
                "concreteType": "NamespaceOutputVisibility",
                "kind": "LinkedField",
                "name": "outputVisibility",
                "plural": false,
                "selections": [
                  {
                    "args": null,
                    "kind": "FragmentSpread",
                    "name": "OutputVisibilitySettingsFormFragment_outputVisibility"
                  }
                ],
                "storageKey": null
              }
            ],
            "storageKey": null
          },
          (v2/*:: as any*/)
        ],
        "storageKey": null
      }
    ],
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*:: as any*/),
    "kind": "Operation",
    "name": "GroupOutputVisibilitySettingsMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*:: as any*/),
        "concreteType": "UpdateGroupPayload",
        "kind": "LinkedField",
        "name": "updateGroup",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "Group",
            "kind": "LinkedField",
            "name": "group",
            "plural": false,
            "selections": [
              {
                "alias": null,
                "args": null,
                "concreteType": "NamespaceOutputVisibility",
                "kind": "LinkedField",
                "name": "outputVisibility",
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
          (v2/*:: as any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "b64eb6a9ea16b3376679a09ee167d4f3",
    "id": null,
    "metadata": {},
    "name": "GroupOutputVisibilitySettingsMutation",
    "operationKind": "mutation",
    "text": "mutation GroupOutputVisibilitySettingsMutation(\n  $input: UpdateGroupInput!\n) {\n  updateGroup(input: $input) {\n    group {\n      outputVisibility {\n        ...OutputVisibilitySettingsFormFragment_outputVisibility\n      }\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment OutputVisibilitySettingsFormFragment_outputVisibility on NamespaceOutputVisibility {\n  inherited\n  namespacePath\n  value\n}\n"
  }
};
})();

(node as any).hash = "a8da0a162b3e3cc9d279ff87add11705";

export default node;
