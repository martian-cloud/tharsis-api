/**
 * @generated SignedSource<<b63e3d7fc111104c255d2c395e011bdb>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ServiceAccountAutocompleteQuery$variables = {
  path: string;
  search: string;
};
export type ServiceAccountAutocompleteQuery$data = {
  readonly namespace: {
    readonly serviceAccounts: {
      readonly edges: ReadonlyArray<{
        readonly node: {
          readonly description: string;
          readonly id: string;
          readonly name: string;
          readonly resourcePath: string;
        } | null | undefined;
      } | null | undefined> | null | undefined;
    };
  } | null | undefined;
};
export type ServiceAccountAutocompleteQuery = {
  response: ServiceAccountAutocompleteQuery$data;
  variables: ServiceAccountAutocompleteQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "path"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "search"
  }
],
v1 = [
  {
    "kind": "Variable",
    "name": "fullPath",
    "variableName": "path"
  }
],
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v3 = {
  "alias": null,
  "args": [
    {
      "kind": "Literal",
      "name": "first",
      "value": 50
    },
    {
      "kind": "Literal",
      "name": "includeInherited",
      "value": true
    },
    {
      "kind": "Variable",
      "name": "search",
      "variableName": "search"
    }
  ],
  "concreteType": "ServiceAccountConnection",
  "kind": "LinkedField",
  "name": "serviceAccounts",
  "plural": false,
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "ServiceAccountEdge",
      "kind": "LinkedField",
      "name": "edges",
      "plural": true,
      "selections": [
        {
          "alias": null,
          "args": null,
          "concreteType": "ServiceAccount",
          "kind": "LinkedField",
          "name": "node",
          "plural": false,
          "selections": [
            (v2/*: any*/),
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
              "name": "resourcePath",
              "storageKey": null
            },
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "description",
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
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "ServiceAccountAutocompleteQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "namespace",
        "plural": false,
        "selections": [
          (v3/*: any*/)
        ],
        "storageKey": null
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "ServiceAccountAutocompleteQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "namespace",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "__typename",
            "storageKey": null
          },
          (v3/*: any*/),
          (v2/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "f3a35e7fffc730e022d0782eb412359b",
    "id": null,
    "metadata": {},
    "name": "ServiceAccountAutocompleteQuery",
    "operationKind": "query",
    "text": "query ServiceAccountAutocompleteQuery(\n  $path: String!\n  $search: String!\n) {\n  namespace(fullPath: $path) {\n    __typename\n    serviceAccounts(first: 50, includeInherited: true, search: $search) {\n      edges {\n        node {\n          id\n          name\n          resourcePath\n          description\n        }\n      }\n    }\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "d073d89fd8c94dbc157f01fef62ab6f5";

export default node;
