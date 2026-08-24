/**
 * @generated SignedSource<<8d28b86e25108fa28031338a58ee4538>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type PackageVersionStatus = "ERRORED" | "PENDING" | "UPLOADED" | "UPLOAD_IN_PROGRESS" | "%future added value";
export type EditPackageVersionQuery$variables = {
  packageId: string;
  versionId: string;
};
export type EditPackageVersionQuery$data = {
  readonly node: {
    readonly id?: string;
    readonly status?: PackageVersionStatus;
    readonly version?: string;
  } | null | undefined;
  readonly pkg: {
    readonly groupPath?: string;
    readonly id?: string;
    readonly name?: string;
  } | null | undefined;
};
export type EditPackageVersionQuery = {
  response: EditPackageVersionQuery$data;
  variables: EditPackageVersionQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "packageId"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "versionId"
  }
],
v1 = [
  {
    "kind": "Variable",
    "name": "id",
    "variableName": "packageId"
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
  "name": "name",
  "storageKey": null
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "groupPath",
  "storageKey": null
},
v5 = [
  {
    "kind": "Variable",
    "name": "id",
    "variableName": "versionId"
  }
],
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "version",
  "storageKey": null
},
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "status",
  "storageKey": null
},
v8 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "__typename",
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "EditPackageVersionQuery",
    "selections": [
      {
        "alias": "pkg",
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
              (v4/*: any*/)
            ],
            "type": "Package",
            "abstractKey": null
          }
        ],
        "storageKey": null
      },
      {
        "alias": null,
        "args": (v5/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          {
            "kind": "InlineFragment",
            "selections": [
              (v2/*: any*/),
              (v6/*: any*/),
              (v7/*: any*/)
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
    "name": "EditPackageVersionQuery",
    "selections": [
      {
        "alias": "pkg",
        "args": (v1/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          (v8/*: any*/),
          (v2/*: any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              (v3/*: any*/),
              (v4/*: any*/)
            ],
            "type": "Package",
            "abstractKey": null
          }
        ],
        "storageKey": null
      },
      {
        "alias": null,
        "args": (v5/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          (v8/*: any*/),
          (v2/*: any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              (v6/*: any*/),
              (v7/*: any*/)
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
    "cacheID": "778c6a31790f876a4287e8bde2d741b2",
    "id": null,
    "metadata": {},
    "name": "EditPackageVersionQuery",
    "operationKind": "query",
    "text": "query EditPackageVersionQuery(\n  $packageId: String!\n  $versionId: String!\n) {\n  pkg: node(id: $packageId) {\n    __typename\n    ... on Package {\n      id\n      name\n      groupPath\n    }\n    id\n  }\n  node(id: $versionId) {\n    __typename\n    ... on PackageVersion {\n      id\n      version\n      status\n    }\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "92f3ab4009c68dbe0aeb598923912234";

export default node;
