/**
 * @generated SignedSource<<5f8cea151b8ff0d866feefdd7f517af2>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type AdminAreaResourceLimitSettingsQuery$variables = Record<PropertyKey, never>;
export type AdminAreaResourceLimitSettingsQuery$data = {
  readonly resourceLimits: ReadonlyArray<{
    readonly id: string;
    readonly metadata: {
      readonly updatedAt: any;
      readonly version: string;
    };
    readonly name: string;
    readonly value: number;
  }>;
};
export type AdminAreaResourceLimitSettingsQuery = {
  response: AdminAreaResourceLimitSettingsQuery$data;
  variables: AdminAreaResourceLimitSettingsQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "alias": null,
    "args": null,
    "concreteType": "ResourceLimit",
    "kind": "LinkedField",
    "name": "resourceLimits",
    "plural": true,
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
  }
];
return {
  "fragment": {
    "argumentDefinitions": [],
    "kind": "Fragment",
    "metadata": null,
    "name": "AdminAreaResourceLimitSettingsQuery",
    "selections": (v0/*: any*/),
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [],
    "kind": "Operation",
    "name": "AdminAreaResourceLimitSettingsQuery",
    "selections": (v0/*: any*/)
  },
  "params": {
    "cacheID": "ea9654ae9f9a971834b8b96148a4318f",
    "id": null,
    "metadata": {},
    "name": "AdminAreaResourceLimitSettingsQuery",
    "operationKind": "query",
    "text": "query AdminAreaResourceLimitSettingsQuery {\n  resourceLimits {\n    id\n    name\n    value\n    metadata {\n      version\n      updatedAt\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "ec43d71fc2d58b3182050c6c60d5e36f";

export default node;
