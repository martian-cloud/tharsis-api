/**
 * @generated SignedSource<<6c20aba88b82408c75ee0845029ef6d7>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type GroupAutocompleteQuery$variables = {
  search: string;
};
export type GroupAutocompleteQuery$data = {
  readonly groups: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly description: string;
        readonly fullPath: string;
        readonly id: string;
        readonly name: string;
      } | null | undefined;
    } | null | undefined> | null | undefined;
  };
};
export type GroupAutocompleteQuery = {
  response: GroupAutocompleteQuery$data;
  variables: GroupAutocompleteQuery$variables;
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
      },
      {
        "kind": "Literal",
        "name": "sort",
        "value": "FULL_PATH_ASC"
      }
    ],
    "concreteType": "GroupConnection",
    "kind": "LinkedField",
    "name": "groups",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "GroupEdge",
        "kind": "LinkedField",
        "name": "edges",
        "plural": true,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "Group",
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
                "name": "fullPath",
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
    "name": "GroupAutocompleteQuery",
    "selections": (v1/*: any*/),
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "GroupAutocompleteQuery",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "ab431e85f6838222ade441d3e5ced303",
    "id": null,
    "metadata": {},
    "name": "GroupAutocompleteQuery",
    "operationKind": "query",
    "text": "query GroupAutocompleteQuery(\n  $search: String!\n) {\n  groups(first: 50, search: $search, sort: FULL_PATH_ASC) {\n    edges {\n      node {\n        id\n        name\n        description\n        fullPath\n      }\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "2dc6297bdc7f90d0bbe5764303029796";

export default node;
