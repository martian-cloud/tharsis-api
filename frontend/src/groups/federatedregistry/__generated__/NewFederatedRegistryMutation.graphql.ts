/**
 * @generated SignedSource<<e2e4e72636e517d69af1a33ad160ea95>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type CreateFederatedRegistryInput = {
  audience: string;
  clientMutationId?: string | null | undefined;
  groupId?: string | null | undefined;
  groupPath?: string | null | undefined;
  hostname: string;
};
export type NewFederatedRegistryMutation$variables = {
  connections: ReadonlyArray<string>;
  input: CreateFederatedRegistryInput;
};
export type NewFederatedRegistryMutation$data = {
  readonly createFederatedRegistry: {
    readonly federatedRegistry: {
      readonly id: string;
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type NewFederatedRegistryMutation = {
  response: NewFederatedRegistryMutation$data;
  variables: NewFederatedRegistryMutation$variables;
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
  "concreteType": "FederatedRegistry",
  "kind": "LinkedField",
  "name": "federatedRegistry",
  "plural": false,
  "selections": [
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
v4 = {
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
    "name": "NewFederatedRegistryMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "FederatedRegistryMutationPayload",
        "kind": "LinkedField",
        "name": "createFederatedRegistry",
        "plural": false,
        "selections": [
          (v3/*: any*/),
          (v4/*: any*/)
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
    "name": "NewFederatedRegistryMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "FederatedRegistryMutationPayload",
        "kind": "LinkedField",
        "name": "createFederatedRegistry",
        "plural": false,
        "selections": [
          (v3/*: any*/),
          {
            "alias": null,
            "args": null,
            "filters": null,
            "handle": "prependNode",
            "key": "",
            "kind": "LinkedHandle",
            "name": "federatedRegistry",
            "handleArgs": [
              {
                "kind": "Variable",
                "name": "connections",
                "variableName": "connections"
              },
              {
                "kind": "Literal",
                "name": "edgeTypeName",
                "value": "FederatedRegistryEdge"
              }
            ]
          },
          (v4/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "bf1c07b27836b93f4f446224e056eab3",
    "id": null,
    "metadata": {},
    "name": "NewFederatedRegistryMutation",
    "operationKind": "mutation",
    "text": "mutation NewFederatedRegistryMutation(\n  $input: CreateFederatedRegistryInput!\n) {\n  createFederatedRegistry(input: $input) {\n    federatedRegistry {\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "5ea52d1f4c8375370b99387d58b8e5a7";

export default node;
