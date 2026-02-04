/**
 * @generated SignedSource<<a965013ea4a66572335fcdca95dd45a5>>
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
export type NamespaceProviderMirrorEnabledInput = {
  enabled?: boolean | null | undefined;
  inherit: boolean;
};
export type NamespaceRunnerTagsInput = {
  inherit: boolean;
  tags?: ReadonlyArray<string> | null | undefined;
};
export type GroupProviderMirrorSettingsMutation$variables = {
  input: UpdateGroupInput;
};
export type GroupProviderMirrorSettingsMutation$data = {
  readonly updateGroup: {
    readonly group: {
      readonly providerMirrorEnabled: {
        readonly " $fragmentSpreads": FragmentRefs<"ProviderMirrorSettingsFormFragment_providerMirrorEnabled">;
      };
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type GroupProviderMirrorSettingsMutation = {
  response: GroupProviderMirrorSettingsMutation$data;
  variables: GroupProviderMirrorSettingsMutation$variables;
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
    "name": "GroupProviderMirrorSettingsMutation",
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
    "name": "GroupProviderMirrorSettingsMutation",
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
    "cacheID": "2cf44cd5f9e4699048a9cfed33a31917",
    "id": null,
    "metadata": {},
    "name": "GroupProviderMirrorSettingsMutation",
    "operationKind": "mutation",
    "text": "mutation GroupProviderMirrorSettingsMutation(\n  $input: UpdateGroupInput!\n) {\n  updateGroup(input: $input) {\n    group {\n      providerMirrorEnabled {\n        ...ProviderMirrorSettingsFormFragment_providerMirrorEnabled\n      }\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment ProviderMirrorSettingsFormFragment_providerMirrorEnabled on NamespaceProviderMirrorEnabled {\n  inherited\n  namespacePath\n  value\n}\n"
  }
};
})();

(node as any).hash = "d114985accab9a91528b7f72ba206b4b";

export default node;
