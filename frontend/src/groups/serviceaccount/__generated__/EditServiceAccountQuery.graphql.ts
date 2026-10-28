/**
 * @generated SignedSource<<397da30d1bcb3c9e777994d49bc8868f>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type BoundClaimsType = "GLOB" | "STRING" | "%future added value";
export type EditServiceAccountQuery$variables = {
  id: string;
};
export type EditServiceAccountQuery$data = {
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
export type EditServiceAccountQuery = {
  response: EditServiceAccountQuery$data;
  variables: EditServiceAccountQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "id"
  }
],
v1 = [
  {
    "kind": "Variable",
    "name": "id",
    "variableName": "id"
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
v10 = [
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
    "name": "EditServiceAccountQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
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
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "EditServiceAccountQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
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
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "type",
                "storageKey": null
              },
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
                "selections": (v10/*: any*/),
                "type": "ServiceAccountActor",
                "abstractKey": null
              },
              {
                "kind": "InlineFragment",
                "selections": (v10/*: any*/),
                "type": "VCSProviderActor",
                "abstractKey": null
              },
              {
                "kind": "InlineFragment",
                "selections": (v10/*: any*/),
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
      }
    ]
  },
  "params": {
    "cacheID": "a3b590e1afd762c83ef049f6a082ff17",
    "id": null,
    "metadata": {},
    "name": "EditServiceAccountQuery",
    "operationKind": "query",
    "text": "query EditServiceAccountQuery(\n  $id: String!\n) {\n  serviceAccount(id: $id) {\n    id\n    name\n    description\n    resourcePath\n    createdBy\n    createdByActor {\n      __typename\n      ...ActorDisplayFragment_actor\n    }\n    clientCredentialsEnabled\n    clientSecretExpiresAt\n    oidcTrustPolicies {\n      issuer\n      boundClaimsType\n      boundClaims {\n        name\n        value\n      }\n    }\n  }\n}\n\nfragment ActorDisplayFragment_actor on Actor {\n  __isActor: __typename\n  type\n  principalId\n  ... on UserActor {\n    user {\n      email\n      id\n    }\n  }\n  ... on ServiceAccountActor {\n    trnSnapshot\n  }\n  ... on VCSProviderActor {\n    trnSnapshot\n  }\n  ... on JobActor {\n    trnSnapshot\n  }\n}\n"
  }
};
})();

(node as any).hash = "3a586f180ea86db9d7b07c5ad2944451";

export default node;
