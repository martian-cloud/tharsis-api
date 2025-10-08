/**
 * @generated SignedSource<<33a9bf7e4ebf601c11616847b77ed18a>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ManagedIdentityAutocompleteQuery$variables = {
  path: string;
  search: string;
};
export type ManagedIdentityAutocompleteQuery$data = {
  readonly namespace: {
    readonly managedIdentities: {
      readonly edges: ReadonlyArray<{
        readonly node: {
          readonly description: string;
          readonly groupPath: string;
          readonly id: string;
          readonly name: string;
          readonly resourcePath: string;
          readonly type: string;
        } | null | undefined;
      } | null | undefined> | null | undefined;
    };
  } | null | undefined;
};
export type ManagedIdentityAutocompleteQuery = {
  response: ManagedIdentityAutocompleteQuery$data;
  variables: ManagedIdentityAutocompleteQuery$variables;
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
    },
    {
      "kind": "Literal",
      "name": "sort",
      "value": "GROUP_LEVEL_DESC"
    }
  ],
  "concreteType": "ManagedIdentityConnection",
  "kind": "LinkedField",
  "name": "managedIdentities",
  "plural": false,
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "ManagedIdentityEdge",
      "kind": "LinkedField",
      "name": "edges",
      "plural": true,
      "selections": [
        {
          "alias": null,
          "args": null,
          "concreteType": "ManagedIdentity",
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
              "name": "groupPath",
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
  ],
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "ManagedIdentityAutocompleteQuery",
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
    "name": "ManagedIdentityAutocompleteQuery",
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
    "cacheID": "3b82da3aaf461a0bf9781f750d44aa60",
    "id": null,
    "metadata": {},
    "name": "ManagedIdentityAutocompleteQuery",
    "operationKind": "query",
    "text": "query ManagedIdentityAutocompleteQuery(\n  $path: String!\n  $search: String!\n) {\n  namespace(fullPath: $path) {\n    __typename\n    managedIdentities(first: 50, includeInherited: true, search: $search, sort: GROUP_LEVEL_DESC) {\n      edges {\n        node {\n          id\n          name\n          groupPath\n          resourcePath\n          description\n          type\n        }\n      }\n    }\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "34e12eb1489e9346e207382fa4a7deb4";

export default node;
