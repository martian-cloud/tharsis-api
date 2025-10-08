/**
 * @generated SignedSource<<213ce4cda33137409b3f77bf16d5a7d5>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type VCSProviderType = "github" | "gitlab" | "%future added value";
export type VCSProviderAutocompleteQuery$variables = {
  path: string;
  search: string;
};
export type VCSProviderAutocompleteQuery$data = {
  readonly workspace: {
    readonly vcsProviders: {
      readonly edges: ReadonlyArray<{
        readonly node: {
          readonly autoCreateWebhooks: boolean;
          readonly description: string;
          readonly id: string;
          readonly name: string;
          readonly type: VCSProviderType;
        } | null | undefined;
      } | null | undefined> | null | undefined;
    };
  } | null | undefined;
};
export type VCSProviderAutocompleteQuery = {
  response: VCSProviderAutocompleteQuery$data;
  variables: VCSProviderAutocompleteQuery$variables;
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
      "value": 100
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
  "concreteType": "VCSProviderConnection",
  "kind": "LinkedField",
  "name": "vcsProviders",
  "plural": false,
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "VCSProviderEdge",
      "kind": "LinkedField",
      "name": "edges",
      "plural": true,
      "selections": [
        {
          "alias": null,
          "args": null,
          "concreteType": "VCSProvider",
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
              "name": "type",
              "storageKey": null
            },
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "autoCreateWebhooks",
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
    "name": "VCSProviderAutocompleteQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "Workspace",
        "kind": "LinkedField",
        "name": "workspace",
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
    "name": "VCSProviderAutocompleteQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "Workspace",
        "kind": "LinkedField",
        "name": "workspace",
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
    "cacheID": "b1655f0601f5ad7803a7bc88d92af235",
    "id": null,
    "metadata": {},
    "name": "VCSProviderAutocompleteQuery",
    "operationKind": "query",
    "text": "query VCSProviderAutocompleteQuery(\n  $path: String!\n  $search: String!\n) {\n  workspace(fullPath: $path) {\n    vcsProviders(first: 100, includeInherited: true, search: $search) {\n      edges {\n        node {\n          id\n          name\n          description\n          type\n          autoCreateWebhooks\n        }\n      }\n    }\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "6c9e57c003a186f5f3956801fc500449";

export default node;
