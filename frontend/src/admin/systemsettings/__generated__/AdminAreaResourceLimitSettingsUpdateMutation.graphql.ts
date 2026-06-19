/**
 * @generated SignedSource<<50a8acd70cf99d7d821db9597741a7f3>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type UpdateResourceLimitInput = {
  clientMutationId?: string | null | undefined;
  metadata?: ResourceMetadataInput | null | undefined;
  name: string;
  value: number;
};
export type ResourceMetadataInput = {
  version: string;
};
export type AdminAreaResourceLimitSettingsUpdateMutation$variables = {
  input: UpdateResourceLimitInput;
};
export type AdminAreaResourceLimitSettingsUpdateMutation$data = {
  readonly updateResourceLimit: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
    }>;
    readonly resourceLimit: {
      readonly id: string;
      readonly metadata: {
        readonly updatedAt: any;
        readonly version: string;
      };
      readonly name: string;
      readonly value: number;
    } | null | undefined;
  };
};
export type AdminAreaResourceLimitSettingsUpdateMutation = {
  response: AdminAreaResourceLimitSettingsUpdateMutation$data;
  variables: AdminAreaResourceLimitSettingsUpdateMutation$variables;
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
    "concreteType": "UpdateResourceLimitPayload",
    "kind": "LinkedField",
    "name": "updateResourceLimit",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "ResourceLimit",
        "kind": "LinkedField",
        "name": "resourceLimit",
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
            "name": "value",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "concreteType": "ResourceMetadata",
            "kind": "LinkedField",
            "name": "metadata",
            "plural": false,
            "selections": [
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "version",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "updatedAt",
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
    "name": "AdminAreaResourceLimitSettingsUpdateMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "AdminAreaResourceLimitSettingsUpdateMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "e1b6b2abebe1e8682a17920ae5eece81",
    "id": null,
    "metadata": {},
    "name": "AdminAreaResourceLimitSettingsUpdateMutation",
    "operationKind": "mutation",
    "text": "mutation AdminAreaResourceLimitSettingsUpdateMutation(\n  $input: UpdateResourceLimitInput!\n) {\n  updateResourceLimit(input: $input) {\n    resourceLimit {\n      id\n      name\n      value\n      metadata {\n        version\n        updatedAt\n      }\n    }\n    problems {\n      message\n      field\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "8cf718316152349e59d11148c5fb590b";

export default node;
