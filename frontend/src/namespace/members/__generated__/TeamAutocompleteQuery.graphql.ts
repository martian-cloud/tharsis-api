/**
 * @generated SignedSource<<10dc0235160819211a18e24debd904a4>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type TeamAutocompleteQuery$variables = {
  search: string;
};
export type TeamAutocompleteQuery$data = {
  readonly teams: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly id: string;
        readonly name: string;
      } | null | undefined;
    } | null | undefined> | null | undefined;
  };
};
export type TeamAutocompleteQuery = {
  response: TeamAutocompleteQuery$data;
  variables: TeamAutocompleteQuery$variables;
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
    "concreteType": "TeamConnection",
    "kind": "LinkedField",
    "name": "teams",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "TeamEdge",
        "kind": "LinkedField",
        "name": "edges",
        "plural": true,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "Team",
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
    "name": "TeamAutocompleteQuery",
    "selections": (v1/*: any*/),
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "TeamAutocompleteQuery",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "461abcbdb57ad9cf38634b5f97dc2132",
    "id": null,
    "metadata": {},
    "name": "TeamAutocompleteQuery",
    "operationKind": "query",
    "text": "query TeamAutocompleteQuery(\n  $search: String!\n) {\n  teams(first: 50, search: $search) {\n    edges {\n      node {\n        id\n        name\n      }\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "f9b9432b2f9523756380728a151fb371";

export default node;
