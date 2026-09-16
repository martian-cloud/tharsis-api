/**
 * @generated SignedSource<<813d3f5b6a1dba3bd6eb59f947d49ac8>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type AdminAreaSCIMTokenSettingsQuery$variables = Record<PropertyKey, never>;
export type AdminAreaSCIMTokenSettingsQuery$data = {
  readonly config: {
    readonly oauthProviders: ReadonlyArray<{
      readonly issuerUrl: string;
    }>;
  };
  readonly scimToken: {
    readonly createdBy: string;
    readonly metadata: {
      readonly createdAt: any;
    };
  } | null | undefined;
};
export type AdminAreaSCIMTokenSettingsQuery = {
  response: AdminAreaSCIMTokenSettingsQuery$data;
  variables: AdminAreaSCIMTokenSettingsQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "createdBy",
  "storageKey": null
},
v1 = {
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
},
v2 = {
  "alias": null,
  "args": null,
  "concreteType": "Config",
  "kind": "LinkedField",
  "name": "config",
  "plural": false,
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "IdpConfig",
      "kind": "LinkedField",
      "name": "oauthProviders",
      "plural": true,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "issuerUrl",
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": [],
    "kind": "Fragment",
    "metadata": null,
    "name": "AdminAreaSCIMTokenSettingsQuery",
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "SCIMToken",
        "kind": "LinkedField",
        "name": "scimToken",
        "plural": false,
        "selections": [
          (v0/*: any*/),
          (v1/*: any*/)
        ],
        "storageKey": null
      },
      (v2/*: any*/)
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [],
    "kind": "Operation",
    "name": "AdminAreaSCIMTokenSettingsQuery",
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "SCIMToken",
        "kind": "LinkedField",
        "name": "scimToken",
        "plural": false,
        "selections": [
          (v0/*: any*/),
          (v1/*: any*/),
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "id",
            "storageKey": null
          }
        ],
        "storageKey": null
      },
      (v2/*: any*/)
    ]
  },
  "params": {
    "cacheID": "7139593a1e7bd9eee35f2db39cd6ad1b",
    "id": null,
    "metadata": {},
    "name": "AdminAreaSCIMTokenSettingsQuery",
    "operationKind": "query",
    "text": "query AdminAreaSCIMTokenSettingsQuery {\n  scimToken {\n    createdBy\n    metadata {\n      createdAt\n    }\n    id\n  }\n  config {\n    oauthProviders {\n      issuerUrl\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "c2a96ecbe568c08c53b1277946b42a70";

export default node;
