/**
 * @generated SignedSource<<ec78868100bd21294e852faa028a98c2>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type EditManagedIdentityQuery$variables = {
  id: string;
};
export type EditManagedIdentityQuery$data = {
  readonly managedIdentity: {
    readonly data: string;
    readonly description: string;
    readonly id: string;
    readonly name: string;
    readonly type: string;
  } | null | undefined;
};
export type EditManagedIdentityQuery = {
  response: EditManagedIdentityQuery$data;
  variables: EditManagedIdentityQuery$variables;
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
    "alias": null,
    "args": [
      {
        "kind": "Variable",
        "name": "id",
        "variableName": "id"
      }
    ],
    "concreteType": "ManagedIdentity",
    "kind": "LinkedField",
    "name": "managedIdentity",
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
        "name": "type",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "name",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "description",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "data",
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
    "name": "EditManagedIdentityQuery",
    "selections": (v1/*: any*/),
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "EditManagedIdentityQuery",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "ee0b9e258309bdba5512fdd9a8c67977",
    "id": null,
    "metadata": {},
    "name": "EditManagedIdentityQuery",
    "operationKind": "query",
    "text": "query EditManagedIdentityQuery(\n  $id: String!\n) {\n  managedIdentity(id: $id) {\n    id\n    type\n    name\n    description\n    data\n  }\n}\n"
  }
};
})();

(node as any).hash = "6f568e451ac7b30c1d18ca07f74e97e9";

export default node;
