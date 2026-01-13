/**
 * @generated SignedSource<<fa8be4a7697aac96292e67de408e6fb2>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type NamespaceType = "GROUP" | "WORKSPACE" | "%future added value";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type NamespaceFavoriteInput = {
  clientMutationId?: string | null | undefined;
  namespacePath: string;
  namespaceType: NamespaceType;
};
export type NamespaceFavoriteButtonDeleteMutation$variables = {
  input: NamespaceFavoriteInput;
};
export type NamespaceFavoriteButtonDeleteMutation$data = {
  readonly unfavoriteNamespace: {
    readonly problems: ReadonlyArray<{
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type NamespaceFavoriteButtonDeleteMutation = {
  response: NamespaceFavoriteButtonDeleteMutation$data;
  variables: NamespaceFavoriteButtonDeleteMutation$variables;
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
    "concreteType": "NamespaceUnfavoriteMutationPayload",
    "kind": "LinkedField",
    "name": "unfavoriteNamespace",
    "plural": false,
    "selections": [
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
    "name": "NamespaceFavoriteButtonDeleteMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "NamespaceFavoriteButtonDeleteMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "e019ff109294b244d7af9ba3764ec89e",
    "id": null,
    "metadata": {},
    "name": "NamespaceFavoriteButtonDeleteMutation",
    "operationKind": "mutation",
    "text": "mutation NamespaceFavoriteButtonDeleteMutation(\n  $input: NamespaceFavoriteInput!\n) {\n  unfavoriteNamespace(input: $input) {\n    problems {\n      message\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "af6ddebd2ff4b8304bfe32be372609a2";

export default node;
