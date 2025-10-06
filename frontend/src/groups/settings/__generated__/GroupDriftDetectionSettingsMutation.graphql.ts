/**
 * @generated SignedSource<<807e2c8d0e47f4ad08e5e73bf37c2741>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type UpdateGroupInput = {
  clientMutationId?: string | null | undefined;
  description?: string | null | undefined;
  driftDetectionEnabled?: NamespaceDriftDetectionEnabledInput | null | undefined;
  groupPath?: string | null | undefined;
  id?: string | null | undefined;
  metadata?: ResourceMetadataInput | null | undefined;
  runnerTags?: NamespaceRunnerTagsInput | null | undefined;
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
export type GroupDriftDetectionSettingsMutation$variables = {
  input: UpdateGroupInput;
};
export type GroupDriftDetectionSettingsMutation$data = {
  readonly updateGroup: {
    readonly group: {
      readonly driftDetectionEnabled: {
        readonly " $fragmentSpreads": FragmentRefs<"DriftDetectionSettingsFormFragment_driftDetectionEnabled">;
      };
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type GroupDriftDetectionSettingsMutation = {
  response: GroupDriftDetectionSettingsMutation$data;
  variables: GroupDriftDetectionSettingsMutation$variables;
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
    "name": "GroupDriftDetectionSettingsMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
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
                "concreteType": "NamespaceDriftDetectionEnabled",
                "kind": "LinkedField",
                "name": "driftDetectionEnabled",
                "plural": false,
                "selections": [
                  {
                    "args": null,
                    "kind": "FragmentSpread",
                    "name": "DriftDetectionSettingsFormFragment_driftDetectionEnabled"
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
    "name": "GroupDriftDetectionSettingsMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
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
                "concreteType": "NamespaceDriftDetectionEnabled",
                "kind": "LinkedField",
                "name": "driftDetectionEnabled",
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
    "cacheID": "e8410333f81ccbd15e06bb28b7829408",
    "id": null,
    "metadata": {},
    "name": "GroupDriftDetectionSettingsMutation",
    "operationKind": "mutation",
    "text": "mutation GroupDriftDetectionSettingsMutation(\n  $input: UpdateGroupInput!\n) {\n  updateGroup(input: $input) {\n    group {\n      driftDetectionEnabled {\n        ...DriftDetectionSettingsFormFragment_driftDetectionEnabled\n      }\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment DriftDetectionSettingsFormFragment_driftDetectionEnabled on NamespaceDriftDetectionEnabled {\n  inherited\n  namespacePath\n  value\n}\n"
  }
};
})();

(node as any).hash = "71fa5ee2d45ae16e6473c6e0c0332760";

export default node;
