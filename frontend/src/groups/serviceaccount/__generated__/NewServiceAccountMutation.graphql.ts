/**
 * @generated SignedSource<<bf929db9568fc7e11145e733f7de075b>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type BoundClaimsType = "GLOB" | "STRING" | "%future added value";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type CreateServiceAccountInput = {
  clientMutationId?: string | null | undefined;
  clientSecretExpiresAt?: any | null | undefined;
  description: string;
  enableClientCredentials: boolean;
  groupId?: string | null | undefined;
  groupPath?: string | null | undefined;
  name: string;
  oidcTrustPolicies: ReadonlyArray<OIDCTrustPolicyInput>;
};
export type OIDCTrustPolicyInput = {
  boundClaims: ReadonlyArray<JWTClaimInput>;
  boundClaimsType?: BoundClaimsType | null | undefined;
  issuer: string;
};
export type JWTClaimInput = {
  name: string;
  value: string;
};
export type NewServiceAccountMutation$variables = {
  input: CreateServiceAccountInput;
};
export type NewServiceAccountMutation$data = {
  readonly createServiceAccount: {
    readonly clientSecret: string | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly serviceAccount: {
      readonly clientCredentialsEnabled: boolean;
      readonly clientSecretExpiresAt: any | null | undefined;
      readonly createdBy: string;
      readonly description: string;
      readonly groupPath: string;
      readonly id: string;
      readonly metadata: {
        readonly updatedAt: any;
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
};
export type NewServiceAccountMutation = {
  response: NewServiceAccountMutation$data;
  variables: NewServiceAccountMutation$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "input"
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
        "name": "input",
        "variableName": "input"
      }
    ],
    "concreteType": "CreateServiceAccountPayload",
    "kind": "LinkedField",
    "name": "createServiceAccount",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "ServiceAccount",
        "kind": "LinkedField",
        "name": "serviceAccount",
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
            "concreteType": "ResourceMetadata",
            "kind": "LinkedField",
            "name": "metadata",
            "plural": false,
            "selections": [
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "updatedAt",
                "storageKey": null
              }
            ],
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
            "name": "groupPath",
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
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "clientSecret",
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
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "type",
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
    "name": "NewServiceAccountMutation",
    "selections": (v2/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "NewServiceAccountMutation",
    "selections": (v2/*: any*/)
  },
  "params": {
    "cacheID": "2de215baa6d0a562280100a6902d9abb",
    "id": null,
    "metadata": {},
    "name": "NewServiceAccountMutation",
    "operationKind": "mutation",
    "text": "mutation NewServiceAccountMutation(\n  $input: CreateServiceAccountInput!\n) {\n  createServiceAccount(input: $input) {\n    serviceAccount {\n      id\n      metadata {\n        updatedAt\n      }\n      name\n      description\n      resourcePath\n      groupPath\n      createdBy\n      clientCredentialsEnabled\n      clientSecretExpiresAt\n      oidcTrustPolicies {\n        issuer\n        boundClaimsType\n        boundClaims {\n          name\n          value\n        }\n      }\n    }\n    clientSecret\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "13a06aad8719af4ead0398c5a7320430";

export default node;
