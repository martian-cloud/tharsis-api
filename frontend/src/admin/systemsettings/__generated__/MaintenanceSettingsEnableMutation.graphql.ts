/**
 * @generated SignedSource<<80622fdaed07975c516305bd5dcb2939>>
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
export type MaintenanceSettingsEnableMutation$variables = {
  input: EnableMaintenanceModeInput;
};
export type MaintenanceSettingsEnableMutation$data = {
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
export type MaintenanceSettingsEnableMutation = {
  response: MaintenanceSettingsEnableMutation$data;
  variables: MaintenanceSettingsEnableMutation$variables;
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
    "name": "MaintenanceSettingsEnableMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "MaintenanceSettingsEnableMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "26eaf9bd52c01e134bb0746787ccfa30",
    "id": null,
    "metadata": {},
    "name": "MaintenanceSettingsEnableMutation",
    "operationKind": "mutation",
    "text": "mutation MaintenanceSettingsEnableMutation(\n  $input: EnableMaintenanceModeInput!\n) {\n  enableMaintenanceMode(input: $input) {\n    maintenanceMode {\n      id\n      createdBy\n      metadata {\n        createdAt\n      }\n    }\n    problems {\n      message\n      field\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "735dc3c753964289041ca74479499dba";

export default node;
