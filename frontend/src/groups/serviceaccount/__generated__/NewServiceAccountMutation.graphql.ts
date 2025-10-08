/**
 * @generated SignedSource<<828443c71229a5f9b19bddf802cb700f>>
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
  description: string;
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
  connections: ReadonlyArray<string>;
  input: CreateServiceAccountInput;
};
export type NewServiceAccountMutation$data = {
  readonly createServiceAccount: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly serviceAccount: {
      readonly createdBy: string;
      readonly description: string;
      readonly id: string;
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
var v0 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "connections"
},
v1 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "input"
},
v2 = [
  {
    "kind": "Variable",
    "name": "input",
    "variableName": "input"
  }
],
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "name",
  "storageKey": null
},
v4 = {
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
    (v3/*: any*/),
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
            (v3/*: any*/),
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
v5 = {
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
};
return {
  "fragment": {
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "NewServiceAccountMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "CreateServiceAccountPayload",
        "kind": "LinkedField",
        "name": "createServiceAccount",
        "plural": false,
        "selections": [
          (v4/*: any*/),
          (v5/*: any*/)
        ],
        "storageKey": null
      }
    ],
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [
      (v1/*: any*/),
      (v0/*: any*/)
    ],
    "kind": "Operation",
    "name": "NewServiceAccountMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "CreateServiceAccountPayload",
        "kind": "LinkedField",
        "name": "createServiceAccount",
        "plural": false,
        "selections": [
          (v4/*: any*/),
          {
            "alias": null,
            "args": null,
            "filters": null,
            "handle": "prependNode",
            "key": "",
            "kind": "LinkedHandle",
            "name": "serviceAccount",
            "handleArgs": [
              {
                "kind": "Variable",
                "name": "connections",
                "variableName": "connections"
              },
              {
                "kind": "Literal",
                "name": "edgeTypeName",
                "value": "ServiceAccountEdge"
              }
            ]
          },
          (v5/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "d3f49dbf8ef31b0fb598c9f937a02f00",
    "id": null,
    "metadata": {},
    "name": "NewServiceAccountMutation",
    "operationKind": "mutation",
    "text": "mutation NewServiceAccountMutation(\n  $input: CreateServiceAccountInput!\n) {\n  createServiceAccount(input: $input) {\n    serviceAccount {\n      id\n      name\n      description\n      resourcePath\n      createdBy\n      oidcTrustPolicies {\n        issuer\n        boundClaimsType\n        boundClaims {\n          name\n          value\n        }\n      }\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "72dc4a5ed07f3b62ffcc18cd91af8c4e";

export default node;
