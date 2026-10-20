/**
 * @generated SignedSource<<bb1ede11c950293c0c0f8d4f1ef35479>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type NamespaceOutputVisibilityLevel = "block_access" | "direct_group_and_subgroups" | "direct_group_only" | "global" | "root_group" | "%future added value";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type CreateGroupInput = {
  clientMutationId?: string | null | undefined;
  description: string;
  driftDetectionEnabled?: NamespaceDriftDetectionEnabledInput | null | undefined;
  name: string;
  outputVisibility?: NamespaceOutputVisibilityInput | null | undefined;
  parentId?: string | null | undefined;
  parentPath?: string | null | undefined;
  providerMirrorEnabled?: NamespaceProviderMirrorEnabledInput | null | undefined;
  runnerTags?: NamespaceRunnerTagsInput | null | undefined;
};
export type NamespaceDriftDetectionEnabledInput = {
  enabled?: boolean | null | undefined;
  inherit: boolean;
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
export type NewGroupMutation$variables = {
  input: CreateGroupInput;
};
export type NewGroupMutation$data = {
  readonly createGroup: {
    readonly group: {
      readonly fullPath: string;
      readonly id: string;
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type NewGroupMutation = {
  response: NewGroupMutation$data;
  variables: NewGroupMutation$variables;
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
    "concreteType": "CreateGroupPayload",
    "kind": "LinkedField",
    "name": "createGroup",
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
    "name": "NewGroupMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "NewGroupMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "ec26c440dc280c66a900a8a33d244831",
    "id": null,
    "metadata": {},
    "name": "NewGroupMutation",
    "operationKind": "mutation",
    "text": "mutation NewGroupMutation(\n  $input: CreateGroupInput!\n) {\n  createGroup(input: $input) {\n    group {\n      id\n      fullPath\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "b48ad076f37ea3cd971589eb1a926529";

export default node;
