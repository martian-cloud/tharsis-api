/**
 * @generated SignedSource<<e67a1bc365290c5b95e9962d8309f57c>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type SensitiveVariableValueQuery$variables = {
  id: string;
  includeSensitiveValue: boolean;
};
export type SensitiveVariableValueQuery$data = {
  readonly namespaceVariableVersion: {
    readonly id: string;
    readonly value: string | null | undefined;
  } | null | undefined;
};
export type SensitiveVariableValueQuery = {
  response: SensitiveVariableValueQuery$data;
  variables: SensitiveVariableValueQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "id"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "includeSensitiveValue"
  }
],
v1 = [
  {
    "alias": null,
    "args": [
      {
        "kind": "Variable",
        "name": "id",
        "variableName": "id"
      },
      {
        "kind": "Variable",
        "name": "includeSensitiveValue",
        "variableName": "includeSensitiveValue"
      }
    ],
    "concreteType": "NamespaceVariableVersion",
    "kind": "LinkedField",
    "name": "namespaceVariableVersion",
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
        "name": "value",
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
    "name": "SensitiveVariableValueQuery",
    "selections": (v1/*: any*/),
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "SensitiveVariableValueQuery",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "221ee73a45728562ebbd75957c521c28",
    "id": null,
    "metadata": {},
    "name": "SensitiveVariableValueQuery",
    "operationKind": "query",
    "text": "query SensitiveVariableValueQuery(\n  $id: String!\n  $includeSensitiveValue: Boolean!\n) {\n  namespaceVariableVersion(id: $id, includeSensitiveValue: $includeSensitiveValue) {\n    id\n    value\n  }\n}\n"
  }
};
})();

(node as any).hash = "a0c71be2b25b742c91f6efcb46a0cfa4";

export default node;
