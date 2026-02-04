/**
 * @generated SignedSource<<206e1296cba5bf30ae76775ba0313439>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ProviderMirrorListQuery$variables = {
  after?: string | null | undefined;
  before?: string | null | undefined;
  first?: number | null | undefined;
  last?: number | null | undefined;
  namespacePath: string;
  search?: string | null | undefined;
};
export type ProviderMirrorListQuery$data = {
  readonly " $fragmentSpreads": FragmentRefs<"ProviderMirrorListFragment_mirrors">;
};
export type ProviderMirrorListQuery = {
  response: ProviderMirrorListQuery$data;
  variables: ProviderMirrorListQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "after"
},
v1 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "before"
},
v2 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "first"
},
v3 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "last"
},
v4 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "namespacePath"
},
v5 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "search"
},
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "__typename",
  "storageKey": null
},
v8 = [
  {
    "kind": "Variable",
    "name": "after",
    "variableName": "after"
  },
  {
    "kind": "Variable",
    "name": "before",
    "variableName": "before"
  },
  {
    "kind": "Variable",
    "name": "first",
    "variableName": "first"
  },
  {
    "kind": "Variable",
    "name": "last",
    "variableName": "last"
  },
  {
    "kind": "Variable",
    "name": "search",
    "variableName": "search"
  },
  {
    "kind": "Literal",
    "name": "sort",
    "value": "TYPE_ASC"
  }
];
return {
  "fragment": {
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/),
      (v2/*: any*/),
      (v3/*: any*/),
      (v4/*: any*/),
      (v5/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "ProviderMirrorListQuery",
    "selections": [
      {
        "args": null,
        "kind": "FragmentSpread",
        "name": "ProviderMirrorListFragment_mirrors"
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [
      (v2/*: any*/),
      (v3/*: any*/),
      (v0/*: any*/),
      (v1/*: any*/),
      (v4/*: any*/),
      (v5/*: any*/)
    ],
    "kind": "Operation",
    "name": "ProviderMirrorListQuery",
    "selections": [
      {
        "alias": null,
        "args": [
          {
            "kind": "Variable",
            "name": "fullPath",
            "variableName": "namespacePath"
          }
        ],
        "concreteType": null,
        "kind": "LinkedField",
        "name": "namespace",
        "plural": false,
        "selections": [
          (v6/*: any*/),
          (v7/*: any*/),
          {
            "alias": null,
            "args": null,
            "concreteType": "NamespaceProviderMirrorEnabled",
            "kind": "LinkedField",
            "name": "providerMirrorEnabled",
            "plural": false,
            "selections": [
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "value",
                "storageKey": null
              }
            ],
            "storageKey": null
          },
          {
            "alias": null,
            "args": (v8/*: any*/),
            "concreteType": "TerraformProviderVersionMirrorConnection",
            "kind": "LinkedField",
            "name": "terraformProviderMirrors",
            "plural": false,
            "selections": [
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "totalCount",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "TerraformProviderVersionMirrorEdge",
                "kind": "LinkedField",
                "name": "edges",
                "plural": true,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "TerraformProviderVersionMirror",
                    "kind": "LinkedField",
                    "name": "node",
                    "plural": false,
                    "selections": [
                      (v6/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "version",
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
                        "name": "providerAddress",
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "ResourceMetadata",
                        "kind": "LinkedField",
                        "name": "metadata",
                        "plural": false,
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "createdAt",
                            "storageKey": null
                          }
                        ],
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "createdBy",
                        "storageKey": null
                      },
                      (v7/*: any*/)
                    ],
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "cursor",
                    "storageKey": null
                  }
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "PageInfo",
                "kind": "LinkedField",
                "name": "pageInfo",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "endCursor",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "hasNextPage",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "hasPreviousPage",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "startCursor",
                    "storageKey": null
                  }
                ],
                "storageKey": null
              }
            ],
            "storageKey": null
          },
          {
            "alias": null,
            "args": (v8/*: any*/),
            "filters": [
              "sort",
              "search"
            ],
            "handle": "connection",
            "key": "ProviderMirrorList_terraformProviderMirrors",
            "kind": "LinkedHandle",
            "name": "terraformProviderMirrors"
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "e8d5540e0e61bfeffa9b2c1df66875e2",
    "id": null,
    "metadata": {},
    "name": "ProviderMirrorListQuery",
    "operationKind": "query",
    "text": "query ProviderMirrorListQuery(\n  $first: Int\n  $last: Int\n  $after: String\n  $before: String\n  $namespacePath: String!\n  $search: String\n) {\n  ...ProviderMirrorListFragment_mirrors\n}\n\nfragment ProviderMirrorListFragment_mirrors on Query {\n  namespace(fullPath: $namespacePath) {\n    id\n    __typename\n    providerMirrorEnabled {\n      value\n    }\n    terraformProviderMirrors(after: $after, before: $before, first: $first, last: $last, sort: TYPE_ASC, search: $search) {\n      totalCount\n      edges {\n        node {\n          id\n          version\n          groupPath\n          providerAddress\n          ...ProviderMirrorListItemFragment_mirror\n          __typename\n        }\n        cursor\n      }\n      pageInfo {\n        endCursor\n        hasNextPage\n        hasPreviousPage\n        startCursor\n      }\n    }\n  }\n}\n\nfragment ProviderMirrorListItemFragment_mirror on TerraformProviderVersionMirror {\n  id\n  metadata {\n    createdAt\n  }\n  version\n  createdBy\n  providerAddress\n}\n"
  }
};
})();

(node as any).hash = "0d2e5d72f6d88b09b1792d6ab978e932";

export default node;
