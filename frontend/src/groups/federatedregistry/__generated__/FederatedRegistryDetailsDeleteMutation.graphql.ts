/**
 * @generated SignedSource<<44b2ff368bd14f411b923eb74adb5aa8>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type DeleteFederatedRegistryInput = {
  clientMutationId?: string | null | undefined;
  id: string;
};
export type FederatedRegistryDetailsDeleteMutation$variables = {
  connections: ReadonlyArray<string>;
  input: DeleteFederatedRegistryInput;
};
export type FederatedRegistryDetailsDeleteMutation$data = {
  readonly deleteFederatedRegistry: {
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
export type FederatedRegistryDetailsDeleteMutation = {
  response: FederatedRegistryDetailsDeleteMutation$data;
  variables: FederatedRegistryDetailsDeleteMutation$variables;
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
  "name": "id",
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
    "name": "FederatedRegistryDetailsDeleteMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "FederatedRegistryMutationPayload",
        "kind": "LinkedField",
        "name": "deleteFederatedRegistry",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "FederatedRegistry",
            "kind": "LinkedField",
            "name": "federatedRegistry",
            "plural": false,
            "selections": [
              (v3/*: any*/)
            ],
            "storageKey": null
          },
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
    "name": "FederatedRegistryDetailsDeleteMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "FederatedRegistryMutationPayload",
        "kind": "LinkedField",
        "name": "deleteFederatedRegistry",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "FederatedRegistry",
            "kind": "LinkedField",
            "name": "federatedRegistry",
            "plural": false,
            "selections": [
              (v3/*: any*/),
              {
                "alias": null,
                "args": null,
                "filters": null,
                "handle": "deleteEdge",
                "key": "",
                "kind": "ScalarHandle",
                "name": "id",
                "handleArgs": [
                  {
                    "kind": "Variable",
                    "name": "connections",
                    "variableName": "connections"
                  }
                ]
              }
            ],
            "storageKey": null
          },
          (v4/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "140f354c9bb4b813c406323012a7b159",
    "id": null,
    "metadata": {},
    "name": "FederatedRegistryDetailsDeleteMutation",
    "operationKind": "mutation",
    "text": "mutation FederatedRegistryDetailsDeleteMutation(\n  $input: DeleteFederatedRegistryInput!\n) {\n  deleteFederatedRegistry(input: $input) {\n    federatedRegistry {\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "330fb60a2f4479660c357bd2ac650ade";

export default node;
