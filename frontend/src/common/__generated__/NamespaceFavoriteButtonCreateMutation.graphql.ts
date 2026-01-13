/**
 * @generated SignedSource<<337405064601e611a02e19d72c5c2f66>>
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
export type NamespaceFavoriteButtonCreateMutation$variables = {
  input: NamespaceFavoriteInput;
};
export type NamespaceFavoriteButtonCreateMutation$data = {
  readonly favoriteNamespace: {
    readonly namespaceFavorite: {
      readonly id: string;
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type NamespaceFavoriteButtonCreateMutation = {
  response: NamespaceFavoriteButtonCreateMutation$data;
  variables: NamespaceFavoriteButtonCreateMutation$variables;
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
    "concreteType": "NamespaceFavoriteMutationPayload",
    "kind": "LinkedField",
    "name": "favoriteNamespace",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "NamespaceFavorite",
        "kind": "LinkedField",
        "name": "namespaceFavorite",
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
    "name": "NamespaceFavoriteButtonCreateMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "NamespaceFavoriteButtonCreateMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "5dbd401a221fe8a0ffd8714213b8b5a5",
    "id": null,
    "metadata": {},
    "name": "NamespaceFavoriteButtonCreateMutation",
    "operationKind": "mutation",
    "text": "mutation NamespaceFavoriteButtonCreateMutation(\n  $input: NamespaceFavoriteInput!\n) {\n  favoriteNamespace(input: $input) {\n    namespaceFavorite {\n      id\n    }\n    problems {\n      message\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "4d09fb7d13ec6fe85e8b1b9e0b104a4e";

export default node;
