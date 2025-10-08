/**
 * @generated SignedSource<<30449c54267671b8bb41b6d837b759e2>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type FederatedRegistryDetailsQuery$variables = {
  id: string;
};
export type FederatedRegistryDetailsQuery$data = {
  readonly node: {
    readonly audience?: string;
    readonly createdBy?: string;
    readonly group?: {
      readonly fullPath: string;
    };
    readonly hostname?: string;
    readonly id?: string;
    readonly metadata?: {
      readonly createdAt: any;
      readonly trn: string;
    };
  } | null | undefined;
};
export type FederatedRegistryDetailsQuery = {
  response: FederatedRegistryDetailsQuery$data;
  variables: FederatedRegistryDetailsQuery$variables;
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
    "kind": "Variable",
    "name": "id",
    "variableName": "id"
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
  "args": null,
  "kind": "ScalarField",
  "name": "hostname",
  "storageKey": null
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "audience",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "createdBy",
  "storageKey": null
},
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "fullPath",
  "storageKey": null
},
v7 = {
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
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "trn",
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
    "name": "FederatedRegistryDetailsQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          {
            "kind": "InlineFragment",
            "selections": [
              (v2/*: any*/),
              (v3/*: any*/),
              (v4/*: any*/),
              (v5/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "Group",
                "kind": "LinkedField",
                "name": "group",
                "plural": false,
                "selections": [
                  (v6/*: any*/)
                ],
                "storageKey": null
              },
              (v7/*: any*/)
            ],
            "type": "FederatedRegistry",
            "abstractKey": null
          }
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
    "name": "FederatedRegistryDetailsQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "__typename",
            "storageKey": null
          },
          (v2/*: any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              (v3/*: any*/),
              (v4/*: any*/),
              (v5/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "Group",
                "kind": "LinkedField",
                "name": "group",
                "plural": false,
                "selections": [
                  (v6/*: any*/),
                  (v2/*: any*/)
                ],
                "storageKey": null
              },
              (v7/*: any*/)
            ],
            "type": "FederatedRegistry",
            "abstractKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "3059a3b5766823f0cdf5729d6bc9de6b",
    "id": null,
    "metadata": {},
    "name": "FederatedRegistryDetailsQuery",
    "operationKind": "query",
    "text": "query FederatedRegistryDetailsQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ... on FederatedRegistry {\n      id\n      hostname\n      audience\n      createdBy\n      group {\n        fullPath\n        id\n      }\n      metadata {\n        createdAt\n        trn\n      }\n    }\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "28b042283608dfd9001108767b038f28";

export default node;
