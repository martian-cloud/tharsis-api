/**
 * @generated SignedSource<<e13c9b702747c68c844eeacba55345c4>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type UpdateTerraformModuleInput = {
  clientMutationId?: string | null | undefined;
  id: string;
  labels?: ReadonlyArray<TerraformModuleLabelInput> | null | undefined;
  metadata?: ResourceMetadataInput | null | undefined;
  private?: boolean | null | undefined;
  repositoryUrl?: string | null | undefined;
};
export type TerraformModuleLabelInput = {
  key: string;
  value: string;
};
export type ResourceMetadataInput = {
  version: string;
};
export type TerraformModuleSettingsUpdateMutation$variables = {
  input: UpdateTerraformModuleInput;
};
export type TerraformModuleSettingsUpdateMutation$data = {
  readonly updateTerraformModule: {
    readonly module: {
      readonly id: string;
      readonly labels: ReadonlyArray<{
        readonly key: string;
        readonly value: string;
      }>;
      readonly private: boolean;
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type TerraformModuleSettingsUpdateMutation = {
  response: TerraformModuleSettingsUpdateMutation$data;
  variables: TerraformModuleSettingsUpdateMutation$variables;
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
    "concreteType": "UpdateTerraformModulePayload",
    "kind": "LinkedField",
    "name": "updateTerraformModule",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "TerraformModule",
        "kind": "LinkedField",
        "name": "module",
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
            "name": "private",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "concreteType": "TerraformModuleLabel",
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
    "name": "TerraformModuleSettingsUpdateMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "TerraformModuleSettingsUpdateMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "2123630a2b355e103597ab7b81151241",
    "id": null,
    "metadata": {},
    "name": "TerraformModuleSettingsUpdateMutation",
    "operationKind": "mutation",
    "text": "mutation TerraformModuleSettingsUpdateMutation(\n  $input: UpdateTerraformModuleInput!\n) {\n  updateTerraformModule(input: $input) {\n    module {\n      id\n      private\n      labels {\n        key\n        value\n      }\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "c5fbf3dd1ef41d6bfc1e550d1092889d";

export default node;
