/**
 * @generated SignedSource<<7a6bfaa1ff7373c42c4a6d4bf6c2fd14>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type BoundClaimsType = "GLOB" | "STRING" | "%future added value";
export type ServiceAccountDetailsQuery$variables = {
  id: string;
};
export type ServiceAccountDetailsQuery$data = {
  readonly serviceAccount: {
    readonly clientCredentialsEnabled: boolean;
    readonly clientSecretExpiresAt: any | null | undefined;
    readonly createdBy: string;
    readonly description: string;
    readonly id: string;
    readonly metadata: {
      readonly createdAt: any;
      readonly trn: string;
    };
    readonly name: string;
    readonly oidcTrustPolicies: ReadonlyArray<{
      readonly boundClaims: ReadonlyArray<{
        readonly name: string;
        readonly value: string;
      }>;
      readonly boundClaimsType: BoundClaimsType;
      readonly issuer: string;
    }>;
    readonly resourcePath: string;
  } | null | undefined;
};
export type ServiceAccountDetailsQuery = {
  response: ServiceAccountDetailsQuery$data;
  variables: ServiceAccountDetailsQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "id"
  }
],
v1 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "name",
  "storageKey": null
},
v2 = [
  {
    "alias": null,
    "args": [
      {
        "kind": "Variable",
        "name": "id",
        "variableName": "id"
      }
    ],
    "concreteType": "ServiceAccount",
    "kind": "LinkedField",
    "name": "serviceAccount",
    "plural": false,
    "selections": [
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
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "trn",
            "storageKey": null
          }
        ],
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "id",
        "storageKey": null
      },
      (v1/*: any*/),
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "description",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "resourcePath",
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
        "kind": "ScalarField",
        "name": "clientCredentialsEnabled",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "clientSecretExpiresAt",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "concreteType": "OIDCTrustPolicy",
        "kind": "LinkedField",
        "name": "oidcTrustPolicies",
        "plural": true,
        "selections": [
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "issuer",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "boundClaimsType",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "concreteType": "JWTClaim",
            "kind": "LinkedField",
            "name": "boundClaims",
            "plural": true,
            "selections": [
              (v1/*: any*/),
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
    "name": "ServiceAccountDetailsQuery",
    "selections": (v2/*: any*/),
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "ServiceAccountDetailsQuery",
    "selections": (v2/*: any*/)
  },
  "params": {
    "cacheID": "1d72781063c2d6ded9a67eee379e1567",
    "id": null,
    "metadata": {},
    "name": "ServiceAccountDetailsQuery",
    "operationKind": "query",
    "text": "query ServiceAccountDetailsQuery(\n  $id: String!\n) {\n  serviceAccount(id: $id) {\n    metadata {\n      createdAt\n      trn\n    }\n    id\n    name\n    description\n    resourcePath\n    createdBy\n    clientCredentialsEnabled\n    clientSecretExpiresAt\n    oidcTrustPolicies {\n      issuer\n      boundClaimsType\n      boundClaims {\n        name\n        value\n      }\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "c891a6fdce90b01c0b5fc37184fb4356";

export default node;
