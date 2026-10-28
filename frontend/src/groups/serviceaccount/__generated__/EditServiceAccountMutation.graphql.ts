/**
 * @generated SignedSource<<ca988a3706e6f9eafe5306d38f056187>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type BoundClaimsType = "GLOB" | "STRING" | "%future added value";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type UpdateServiceAccountInput = {
  clientMutationId?: string | null | undefined;
  clientSecretExpiresAt?: any | null | undefined;
  description: string;
  enableClientCredentials?: boolean | null | undefined;
  id: string;
  metadata?: ResourceMetadataInput | null | undefined;
  oidcTrustPolicies: ReadonlyArray<OIDCTrustPolicyInput>;
};
export type ResourceMetadataInput = {
  version: string;
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
export type EditServiceAccountMutation$variables = {
  input: UpdateServiceAccountInput;
};
export type EditServiceAccountMutation$data = {
  readonly updateServiceAccount: {
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
      readonly createdByActor: {
        readonly " $fragmentSpreads": FragmentRefs<"ActorDisplayFragment_actor">;
      } | null | undefined;
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
export type EditServiceAccountMutation = {
  response: EditServiceAccountMutation$data;
  variables: EditServiceAccountMutation$variables;
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
    "kind": "Variable",
    "name": "input",
    "variableName": "input"
  }
],
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
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
  "kind": "ScalarField",
  "name": "description",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "resourcePath",
  "storageKey": null
},
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "createdBy",
  "storageKey": null
},
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "clientCredentialsEnabled",
  "storageKey": null
},
v8 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "clientSecretExpiresAt",
  "storageKey": null
},
v9 = {
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
},
v10 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "clientSecret",
  "storageKey": null
},
v11 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "type",
  "storageKey": null
},
v12 = {
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
    (v11/*: any*/)
  ],
  "storageKey": null
},
v13 = [
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "trnSnapshot",
    "storageKey": null
  }
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "EditServiceAccountMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "UpdateServiceAccountPayload",
        "kind": "LinkedField",
        "name": "updateServiceAccount",
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
              (v2/*: any*/),
              (v3/*: any*/),
              (v4/*: any*/),
              (v5/*: any*/),
              (v6/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": null,
                "kind": "LinkedField",
                "name": "createdByActor",
                "plural": false,
                "selections": [
                  {
                    "args": null,
                    "kind": "FragmentSpread",
                    "name": "ActorDisplayFragment_actor"
                  }
                ],
                "storageKey": null
              },
              (v7/*: any*/),
              (v8/*: any*/),
              (v9/*: any*/)
            ],
            "storageKey": null
          },
          (v10/*: any*/),
          (v12/*: any*/)
        ],
        "storageKey": null
      }
    ],
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "EditServiceAccountMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "UpdateServiceAccountPayload",
        "kind": "LinkedField",
        "name": "updateServiceAccount",
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
              (v2/*: any*/),
              (v3/*: any*/),
              (v4/*: any*/),
              (v5/*: any*/),
              (v6/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": null,
                "kind": "LinkedField",
                "name": "createdByActor",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "__typename",
                    "storageKey": null
                  },
                  {
                    "kind": "TypeDiscriminator",
                    "abstractKey": "__isActor"
                  },
                  (v11/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "principalId",
                    "storageKey": null
                  },
                  {
                    "kind": "InlineFragment",
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "User",
                        "kind": "LinkedField",
                        "name": "user",
                        "plural": false,
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "email",
                            "storageKey": null
                          },
                          (v2/*: any*/)
                        ],
                        "storageKey": null
                      }
                    ],
                    "type": "UserActor",
                    "abstractKey": null
                  },
                  {
                    "kind": "InlineFragment",
                    "selections": (v13/*: any*/),
                    "type": "ServiceAccountActor",
                    "abstractKey": null
                  },
                  {
                    "kind": "InlineFragment",
                    "selections": (v13/*: any*/),
                    "type": "VCSProviderActor",
                    "abstractKey": null
                  },
                  {
                    "kind": "InlineFragment",
                    "selections": (v13/*: any*/),
                    "type": "JobActor",
                    "abstractKey": null
                  }
                ],
                "storageKey": null
              },
              (v7/*: any*/),
              (v8/*: any*/),
              (v9/*: any*/)
            ],
            "storageKey": null
          },
          (v10/*: any*/),
          (v12/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "2afb72717f82e902b1a94c83533a1a1f",
    "id": null,
    "metadata": {},
    "name": "EditServiceAccountMutation",
    "operationKind": "mutation",
    "text": "mutation EditServiceAccountMutation(\n  $input: UpdateServiceAccountInput!\n) {\n  updateServiceAccount(input: $input) {\n    serviceAccount {\n      id\n      name\n      description\n      resourcePath\n      createdBy\n      createdByActor {\n        __typename\n        ...ActorDisplayFragment_actor\n      }\n      clientCredentialsEnabled\n      clientSecretExpiresAt\n      oidcTrustPolicies {\n        issuer\n        boundClaimsType\n        boundClaims {\n          name\n          value\n        }\n      }\n    }\n    clientSecret\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment ActorDisplayFragment_actor on Actor {\n  __isActor: __typename\n  type\n  principalId\n  ... on UserActor {\n    user {\n      email\n      id\n    }\n  }\n  ... on ServiceAccountActor {\n    trnSnapshot\n  }\n  ... on VCSProviderActor {\n    trnSnapshot\n  }\n  ... on JobActor {\n    trnSnapshot\n  }\n}\n"
  }
};
})();

(node as any).hash = "57958c446856ccd7afa2d9caf59bb2b9";

export default node;
