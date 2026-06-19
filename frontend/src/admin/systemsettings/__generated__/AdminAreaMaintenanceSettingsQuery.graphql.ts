/**
 * @generated SignedSource<<ff6289e5f2dea9e48879087f6d32520d>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type AdminAreaMaintenanceSettingsQuery$variables = Record<PropertyKey, never>;
export type AdminAreaMaintenanceSettingsQuery$data = {
  readonly maintenanceMode: {
    readonly createdBy: string;
    readonly id: string;
    readonly metadata: {
      readonly createdAt: any;
    };
  } | null | undefined;
};
export type AdminAreaMaintenanceSettingsQuery = {
  response: AdminAreaMaintenanceSettingsQuery$data;
  variables: AdminAreaMaintenanceSettingsQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
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
  }
];
return {
  "fragment": {
    "argumentDefinitions": [],
    "kind": "Fragment",
    "metadata": null,
    "name": "AdminAreaMaintenanceSettingsQuery",
    "selections": (v0/*: any*/),
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [],
    "kind": "Operation",
    "name": "AdminAreaMaintenanceSettingsQuery",
    "selections": (v0/*: any*/)
  },
  "params": {
    "cacheID": "a84303f2454744785ab5bdc4713d3e29",
    "id": null,
    "metadata": {},
    "name": "AdminAreaMaintenanceSettingsQuery",
    "operationKind": "query",
    "text": "query AdminAreaMaintenanceSettingsQuery {\n  maintenanceMode {\n    id\n    createdBy\n    metadata {\n      createdAt\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "862c49c3b7185805288e5975eb70f2b2";

export default node;
