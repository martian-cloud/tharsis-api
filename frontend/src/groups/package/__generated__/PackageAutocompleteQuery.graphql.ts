/**
 * @generated SignedSource<<cf3916df9d6771680aa9d67da3929cec>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type PackageVisibility = "GLOBAL" | "PRIVATE" | "ROOT_GROUP" | "%future added value";
export type PackageAutocompleteQuery$variables = {
  first?: number | null | undefined;
  fullPath: string;
  search: string;
};
export type PackageAutocompleteQuery$data = {
  readonly group: {
    readonly visiblePackages: {
      readonly edges: ReadonlyArray<{
        readonly node: {
          readonly description: string;
          readonly groupPath: string;
          readonly id: string;
          readonly name: string;
          readonly visibility: PackageVisibility;
        } | null | undefined;
      } | null | undefined> | null | undefined;
    };
  } | null | undefined;
};
export type PackageAutocompleteQuery = {
  response: PackageAutocompleteQuery$data;
  variables: PackageAutocompleteQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "first"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "fullPath"
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
    "variableName": "fullPath"
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
      "kind": "Variable",
      "name": "first",
      "variableName": "first"
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
  "concreteType": "PackageConnection",
  "kind": "LinkedField",
  "name": "visiblePackages",
  "plural": false,
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "PackageEdge",
      "kind": "LinkedField",
      "name": "edges",
      "plural": true,
      "selections": [
        {
          "alias": null,
          "args": null,
          "concreteType": "Package",
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
              "name": "description",
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
              "name": "visibility",
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
    "name": "PackageAutocompleteQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "Group",
        "kind": "LinkedField",
        "name": "group",
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
    "name": "PackageAutocompleteQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "Group",
        "kind": "LinkedField",
        "name": "group",
        "plural": false,
        "selections": [
          (v3/*: any*/),
          (v2/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "7f953b865f9bf7150d9cad62d04ddf9f",
    "id": null,
    "metadata": {},
    "name": "PackageAutocompleteQuery",
    "operationKind": "query",
    "text": "query PackageAutocompleteQuery(\n  $first: Int\n  $fullPath: String!\n  $search: String!\n) {\n  group(fullPath: $fullPath) {\n    visiblePackages(first: $first, search: $search, sort: GROUP_LEVEL_DESC) {\n      edges {\n        node {\n          id\n          name\n          description\n          groupPath\n          visibility\n        }\n      }\n    }\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "d9e650de5ea04f8f06b5b6a586e5186f";

export default node;
