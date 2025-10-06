/**
 * @generated SignedSource<<b15c5a1d2bf132afc2b99151024bcd86>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type UpdateFederatedRegistryInput = {
  audience?: string | null | undefined;
  clientMutationId?: string | null | undefined;
  hostname?: string | null | undefined;
  id: string;
};
export type EditFederatedRegistryMutation$variables = {
  input: UpdateFederatedRegistryInput;
};
export type EditFederatedRegistryMutation$data = {
  readonly updateFederatedRegistry: {
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
export type EditFederatedRegistryMutation = {
  response: EditFederatedRegistryMutation$data;
  variables: EditFederatedRegistryMutation$variables;
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
    "concreteType": "FederatedRegistryMutationPayload",
    "kind": "LinkedField",
    "name": "updateFederatedRegistry",
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
    "name": "EditFederatedRegistryMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "EditFederatedRegistryMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "8b31d786c13b9cc8534dc28577b5b04f",
    "id": null,
    "metadata": {},
    "name": "EditFederatedRegistryMutation",
    "operationKind": "mutation",
    "text": "mutation EditFederatedRegistryMutation(\n  $input: UpdateFederatedRegistryInput!\n) {\n  updateFederatedRegistry(input: $input) {\n    federatedRegistry {\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "1d2f2c628a7debbf590bebfc7c854699";

export default node;
