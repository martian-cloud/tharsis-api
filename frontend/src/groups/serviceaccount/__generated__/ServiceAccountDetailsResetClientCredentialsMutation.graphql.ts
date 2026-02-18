/**
 * @generated SignedSource<<f86f2f4d72d0365bdeb4f3a37cb943fb>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type ResetServiceAccountClientCredentialsInput = {
  clientMutationId?: string | null | undefined;
  clientSecretExpiresAt?: any | null | undefined;
  id: string;
};
export type ServiceAccountDetailsResetClientCredentialsMutation$variables = {
  input: ResetServiceAccountClientCredentialsInput;
};
export type ServiceAccountDetailsResetClientCredentialsMutation$data = {
  readonly resetServiceAccountClientCredentials: {
    readonly clientSecret: string | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly serviceAccount: {
      readonly clientCredentialsEnabled: boolean;
      readonly clientSecretExpiresAt: any | null | undefined;
      readonly id: string;
    } | null | undefined;
  };
};
export type ServiceAccountDetailsResetClientCredentialsMutation = {
  response: ServiceAccountDetailsResetClientCredentialsMutation$data;
  variables: ServiceAccountDetailsResetClientCredentialsMutation$variables;
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
    "concreteType": "ResetServiceAccountClientCredentialsPayload",
    "kind": "LinkedField",
    "name": "resetServiceAccountClientCredentials",
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
    "name": "ServiceAccountDetailsResetClientCredentialsMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "ServiceAccountDetailsResetClientCredentialsMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "be3ff73bee3d3c71004ab4ba28b19d71",
    "id": null,
    "metadata": {},
    "name": "ServiceAccountDetailsResetClientCredentialsMutation",
    "operationKind": "mutation",
    "text": "mutation ServiceAccountDetailsResetClientCredentialsMutation(\n  $input: ResetServiceAccountClientCredentialsInput!\n) {\n  resetServiceAccountClientCredentials(input: $input) {\n    serviceAccount {\n      id\n      clientCredentialsEnabled\n      clientSecretExpiresAt\n    }\n    clientSecret\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "5ebb0e9949170a48e0015b95f7242a4f";

export default node;
