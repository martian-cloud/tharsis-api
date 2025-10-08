/**
 * @generated SignedSource<<9fadc05cc689dfc23e32520689300858>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type PrincipalAutocompleteQuery$variables = {
  first?: number | null | undefined;
  fullPath: string;
  search: string;
};
export type PrincipalAutocompleteQuery$data = {
  readonly group: {
    readonly serviceAccounts: {
      readonly edges: ReadonlyArray<{
        readonly node: {
          readonly id: string;
          readonly name: string;
          readonly resourcePath: string;
        } | null | undefined;
      } | null | undefined> | null | undefined;
    };
  } | null | undefined;
  readonly teams: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly id: string;
        readonly name: string;
      } | null | undefined;
    } | null | undefined> | null | undefined;
  };
  readonly users: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly email: string;
        readonly id: string;
        readonly username: string;
      } | null | undefined;
    } | null | undefined> | null | undefined;
  };
};
export type PrincipalAutocompleteQuery = {
  response: PrincipalAutocompleteQuery$data;
  variables: PrincipalAutocompleteQuery$variables;
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
  "kind": "Variable",
  "name": "first",
  "variableName": "first"
},
v3 = {
  "kind": "Variable",
  "name": "search",
  "variableName": "search"
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "name",
  "storageKey": null
},
v6 = {
  "alias": null,
  "args": [
    (v2/*: any*/),
    {
      "kind": "Literal",
      "name": "includeInherited",
      "value": true
    },
    (v3/*: any*/)
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
            (v4/*: any*/),
            (v5/*: any*/),
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "resourcePath",
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
},
v7 = [
  (v2/*: any*/),
  (v3/*: any*/)
],
v8 = {
  "alias": null,
  "args": (v7/*: any*/),
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
            (v4/*: any*/),
            (v5/*: any*/)
          ],
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "storageKey": null
},
v9 = {
  "alias": null,
  "args": (v7/*: any*/),
  "concreteType": "UserConnection",
  "kind": "LinkedField",
  "name": "users",
  "plural": false,
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "UserEdge",
      "kind": "LinkedField",
      "name": "edges",
      "plural": true,
      "selections": [
        {
          "alias": null,
          "args": null,
          "concreteType": "User",
          "kind": "LinkedField",
          "name": "node",
          "plural": false,
          "selections": [
            (v4/*: any*/),
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "username",
              "storageKey": null
            },
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "email",
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
    "name": "PrincipalAutocompleteQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "Group",
        "kind": "LinkedField",
        "name": "group",
        "plural": false,
        "selections": [
          (v6/*: any*/)
        ],
        "storageKey": null
      },
      (v8/*: any*/),
      (v9/*: any*/)
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "PrincipalAutocompleteQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "Group",
        "kind": "LinkedField",
        "name": "group",
        "plural": false,
        "selections": [
          (v6/*: any*/),
          (v4/*: any*/)
        ],
        "storageKey": null
      },
      (v8/*: any*/),
      (v9/*: any*/)
    ]
  },
  "params": {
    "cacheID": "258158ec36a993cae46ddc041385d8cc",
    "id": null,
    "metadata": {},
    "name": "PrincipalAutocompleteQuery",
    "operationKind": "query",
    "text": "query PrincipalAutocompleteQuery(\n  $first: Int\n  $fullPath: String!\n  $search: String!\n) {\n  group(fullPath: $fullPath) {\n    serviceAccounts(first: $first, search: $search, includeInherited: true) {\n      edges {\n        node {\n          id\n          name\n          resourcePath\n        }\n      }\n    }\n    id\n  }\n  teams(first: $first, search: $search) {\n    edges {\n      node {\n        id\n        name\n      }\n    }\n  }\n  users(first: $first, search: $search) {\n    edges {\n      node {\n        id\n        username\n        email\n      }\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "fe35d6c97d8c19ad1d3c2ef77f331da1";

export default node;
