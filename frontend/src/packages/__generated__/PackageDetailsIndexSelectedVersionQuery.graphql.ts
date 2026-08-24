/**
 * @generated SignedSource<<ef62c31162e8257cc23bfd1cc2ddb04a>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type PackageDetailsIndexSelectedVersionQuery$variables = {
  id: string;
};
export type PackageDetailsIndexSelectedVersionQuery$data = {
  readonly node: {
    readonly " $fragmentSpreads": FragmentRefs<"PackageDetailsIndexFragment_version">;
  } | null | undefined;
};
export type PackageDetailsIndexSelectedVersionQuery = {
  response: PackageDetailsIndexSelectedVersionQuery$data;
  variables: PackageDetailsIndexSelectedVersionQuery$variables;
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
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "PackageDetailsIndexSelectedVersionQuery",
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
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "PackageDetailsIndexFragment_version"
              }
            ],
            "type": "PackageVersion",
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
    "name": "PackageDetailsIndexSelectedVersionQuery",
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
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "id",
            "storageKey": null
          },
          {
            "kind": "InlineFragment",
            "selections": [
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
                "name": "status",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "error",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "latest",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "createdBy",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "shaSum",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "size",
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
              }
            ],
            "type": "PackageVersion",
            "abstractKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "0b5ed896b4ccfb407ff6396205ec4b63",
    "id": null,
    "metadata": {},
    "name": "PackageDetailsIndexSelectedVersionQuery",
    "operationKind": "query",
    "text": "query PackageDetailsIndexSelectedVersionQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ... on PackageVersion {\n      ...PackageDetailsIndexFragment_version\n    }\n    id\n  }\n}\n\nfragment PackageDetailsIndexFragment_version on PackageVersion {\n  id\n  version\n  status\n  error\n  latest\n  ...PackageDetailsSidebarFragment_version\n}\n\nfragment PackageDetailsSidebarFragment_version on PackageVersion {\n  version\n  latest\n  createdBy\n  shaSum\n  size\n  metadata {\n    createdAt\n  }\n}\n"
  }
};
})();

(node as any).hash = "a53fc2c453e1d0ede175a88b2c01cea8";

export default node;
