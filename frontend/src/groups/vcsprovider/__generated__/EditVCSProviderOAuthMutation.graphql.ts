/**
 * @generated SignedSource<<03e2d25a5141c66ebec3606bd73f2283>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest, Mutation } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "%future added value";
export type UpdateVCSProviderInput = {
  clientMutationId?: string | null;
  description?: string | null;
  id: string;
  metadata?: ResourceMetadataInput | null;
  oAuthClientId?: string | null;
  oAuthClientSecret?: string | null;
};
export type ResourceMetadataInput = {
  version: string;
};
export type EditVCSProviderOAuthMutation$variables = {
  input: UpdateVCSProviderInput;
};
export type EditVCSProviderOAuthMutation$data = {
  readonly updateVCSProvider: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly vcsProvider: {
      readonly id: string;
      readonly name: string;
    } | null;
  };
};
export type EditVCSProviderOAuthMutation = {
  response: EditVCSProviderOAuthMutation$data;
  variables: EditVCSProviderOAuthMutation$variables;
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
    "concreteType": "UpdateVCSProviderPayload",
    "kind": "LinkedField",
    "name": "updateVCSProvider",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "VCSProvider",
        "kind": "LinkedField",
        "name": "vcsProvider",
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
            "name": "name",
            "storageKey": null
          }
        ],
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
    "name": "EditVCSProviderOAuthMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "EditVCSProviderOAuthMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "706edcd513b8f169bec81c75220a3308",
    "id": null,
    "metadata": {},
    "name": "EditVCSProviderOAuthMutation",
    "operationKind": "mutation",
    "text": "mutation EditVCSProviderOAuthMutation(\n  $input: UpdateVCSProviderInput!\n) {\n  updateVCSProvider(input: $input) {\n    vcsProvider {\n      id\n      name\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "e89d5ceb06cb3b0efc3de92dd8095270";

export default node;
