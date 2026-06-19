/**
 * @generated SignedSource<<fe38e6a9d1f49412fa3e6cd03a8270b4>>
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
export type AdminAreaMaintenanceSettingsDisableMutation$variables = {
  input: DisableMaintenanceModeInput;
};
export type AdminAreaMaintenanceSettingsDisableMutation$data = {
  readonly disableMaintenanceMode: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
    }>;
  };
};
export type AdminAreaMaintenanceSettingsDisableMutation = {
  response: AdminAreaMaintenanceSettingsDisableMutation$data;
  variables: AdminAreaMaintenanceSettingsDisableMutation$variables;
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
    "name": "AdminAreaMaintenanceSettingsDisableMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "AdminAreaMaintenanceSettingsDisableMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "83ee7bfc76e8a74ccf9ca59720a1af9d",
    "id": null,
    "metadata": {},
    "name": "AdminAreaMaintenanceSettingsDisableMutation",
    "operationKind": "mutation",
    "text": "mutation AdminAreaMaintenanceSettingsDisableMutation(\n  $input: DisableMaintenanceModeInput!\n) {\n  disableMaintenanceMode(input: $input) {\n    problems {\n      message\n      field\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "86091fd8d90082217bf559bb0c8fa383";

export default node;
