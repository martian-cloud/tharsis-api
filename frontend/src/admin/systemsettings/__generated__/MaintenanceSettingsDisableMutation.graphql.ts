/**
 * @generated SignedSource<<84a7b503b4c0c054d68432d7c77b57cd>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type DisableMaintenanceModeInput = {
  clientMutationId?: string | null | undefined;
};
export type MaintenanceSettingsDisableMutation$variables = {
  input: DisableMaintenanceModeInput;
};
export type MaintenanceSettingsDisableMutation$data = {
  readonly disableMaintenanceMode: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
    }>;
  };
};
export type MaintenanceSettingsDisableMutation = {
  response: MaintenanceSettingsDisableMutation$data;
  variables: MaintenanceSettingsDisableMutation$variables;
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
    "concreteType": "DisableMaintenanceModePayload",
    "kind": "LinkedField",
    "name": "disableMaintenanceMode",
    "plural": false,
    "selections": [
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
    "name": "MaintenanceSettingsDisableMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "MaintenanceSettingsDisableMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "38af89a8c4359d95e80c63c8edf2d664",
    "id": null,
    "metadata": {},
    "name": "MaintenanceSettingsDisableMutation",
    "operationKind": "mutation",
    "text": "mutation MaintenanceSettingsDisableMutation(\n  $input: DisableMaintenanceModeInput!\n) {\n  disableMaintenanceMode(input: $input) {\n    problems {\n      message\n      field\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "46561c41664cddc4d35b6ff0fc88fe35";

export default node;
