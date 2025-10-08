/**
 * @generated SignedSource<<ce28b44f8dad56ae5e4575f5948c993a>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type MaintenanceSettingsQuery$variables = Record<PropertyKey, never>;
export type MaintenanceSettingsQuery$data = {
  readonly maintenanceMode: {
    readonly createdBy: string;
    readonly id: string;
    readonly metadata: {
      readonly createdAt: any;
    };
  } | null | undefined;
};
export type MaintenanceSettingsQuery = {
  response: MaintenanceSettingsQuery$data;
  variables: MaintenanceSettingsQuery$variables;
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
    "name": "MaintenanceSettingsQuery",
    "selections": (v0/*: any*/),
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [],
    "kind": "Operation",
    "name": "MaintenanceSettingsQuery",
    "selections": (v0/*: any*/)
  },
  "params": {
    "cacheID": "8c60b698feb528e6a4e1238f245d5843",
    "id": null,
    "metadata": {},
    "name": "MaintenanceSettingsQuery",
    "operationKind": "query",
    "text": "query MaintenanceSettingsQuery {\n  maintenanceMode {\n    id\n    createdBy\n    metadata {\n      createdAt\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "ba894116d687388b94327791799f20e9";

export default node;
