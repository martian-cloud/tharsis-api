/**
 * @generated SignedSource<<1749a0f528923cce35e95a0ca51a2ea6>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type EnableMaintenanceModeInput = {
  clientMutationId?: string | null | undefined;
};
export type AdminAreaMaintenanceSettingsEnableMutation$variables = {
  input: EnableMaintenanceModeInput;
};
export type AdminAreaMaintenanceSettingsEnableMutation$data = {
  readonly enableMaintenanceMode: {
    readonly maintenanceMode: {
      readonly createdBy: string;
      readonly id: string;
      readonly metadata: {
        readonly createdAt: any;
      };
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
    }>;
  };
};
export type AdminAreaMaintenanceSettingsEnableMutation = {
  response: AdminAreaMaintenanceSettingsEnableMutation$data;
  variables: AdminAreaMaintenanceSettingsEnableMutation$variables;
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
    "concreteType": "EnableMaintenanceModePayload",
    "kind": "LinkedField",
    "name": "enableMaintenanceMode",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "MaintenanceMode",
        "kind": "LinkedField",
        "name": "maintenanceMode",
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
            "name": "createdBy",
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
                "name": "createdAt",
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
    "name": "AdminAreaMaintenanceSettingsEnableMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "AdminAreaMaintenanceSettingsEnableMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "1aa403e368ec9c27bec7e5abccdedde5",
    "id": null,
    "metadata": {},
    "name": "AdminAreaMaintenanceSettingsEnableMutation",
    "operationKind": "mutation",
    "text": "mutation AdminAreaMaintenanceSettingsEnableMutation(\n  $input: EnableMaintenanceModeInput!\n) {\n  enableMaintenanceMode(input: $input) {\n    maintenanceMode {\n      id\n      createdBy\n      metadata {\n        createdAt\n      }\n    }\n    problems {\n      message\n      field\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "afa4b4e8ea34afedb80c288d931cd053";

export default node;
