/**
 * @generated SignedSource<<b652635f0b2621d4fc9fe8b5bd1a075e>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type CleanupRuleKind = "RUNS" | "TERRAFORM_MODULES" | "TERRAFORM_PROVIDERS" | "%future added value";
export type NewCleanupPolicyQuery$variables = {
  namespacePath: string;
};
export type NewCleanupPolicyQuery$data = {
  readonly namespace: {
    readonly __typename: string;
    readonly effectiveCleanupPolicies: ReadonlyArray<{
      readonly id: string;
      readonly kind: CleanupRuleKind;
      readonly namespacePath: string;
    }>;
    readonly fullPath: string;
    readonly id: string;
  } | null | undefined;
};
export type NewCleanupPolicyQuery = {
  response: NewCleanupPolicyQuery$data;
  variables: NewCleanupPolicyQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "namespacePath"
  }
],
v1 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v2 = [
  {
    "alias": null,
    "args": [
      {
        "kind": "Variable",
        "name": "fullPath",
        "variableName": "namespacePath"
      }
    ],
    "concreteType": null,
    "kind": "LinkedField",
    "name": "namespace",
    "plural": false,
    "selections": [
      (v1/*: any*/),
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "__typename",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "fullPath",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "concreteType": "CleanupPolicy",
        "kind": "LinkedField",
        "name": "effectiveCleanupPolicies",
        "plural": true,
        "selections": [
          (v1/*: any*/),
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "kind",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "namespacePath",
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
    "name": "NewCleanupPolicyQuery",
    "selections": (v2/*: any*/),
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "NewCleanupPolicyQuery",
    "selections": (v2/*: any*/)
  },
  "params": {
    "cacheID": "4739aa6053554bdb2fe9a86996a9da5a",
    "id": null,
    "metadata": {},
    "name": "NewCleanupPolicyQuery",
    "operationKind": "query",
    "text": "query NewCleanupPolicyQuery(\n  $namespacePath: String!\n) {\n  namespace(fullPath: $namespacePath) {\n    id\n    __typename\n    fullPath\n    effectiveCleanupPolicies {\n      id\n      kind\n      namespacePath\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "5251a0699b00e1493b4942f65bb8da89";

export default node;
