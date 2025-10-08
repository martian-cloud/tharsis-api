/**
 * @generated SignedSource<<783788712329f7bdf267b4309d01857e>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type RoleAutocompleteQuery$variables = {
  search: string;
};
export type RoleAutocompleteQuery$data = {
  readonly roles: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly id: string;
        readonly name: string;
      } | null | undefined;
    } | null | undefined> | null | undefined;
  };
};
export type RoleAutocompleteQuery = {
  response: RoleAutocompleteQuery$data;
  variables: RoleAutocompleteQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "search"
  }
],
v1 = [
  {
    "alias": null,
    "args": [
      {
        "kind": "Literal",
        "name": "first",
        "value": 50
      },
      {
        "kind": "Variable",
        "name": "search",
        "variableName": "search"
      }
    ],
    "concreteType": "RoleConnection",
    "kind": "LinkedField",
    "name": "roles",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "RoleEdge",
        "kind": "LinkedField",
        "name": "edges",
        "plural": true,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "Role",
            "kind": "LinkedField",
            "name": "node",
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
    "name": "RoleAutocompleteQuery",
    "selections": (v1/*: any*/),
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "RoleAutocompleteQuery",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "58a19abdac51ae201bb2c2412c27fb6f",
    "id": null,
    "metadata": {},
    "name": "RoleAutocompleteQuery",
    "operationKind": "query",
    "text": "query RoleAutocompleteQuery(\n  $search: String!\n) {\n  roles(first: 50, search: $search) {\n    edges {\n      node {\n        id\n        name\n      }\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "0c94924ac958f30d82c32532aaf4ca87";

export default node;
