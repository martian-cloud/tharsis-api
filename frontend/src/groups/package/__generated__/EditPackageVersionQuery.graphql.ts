/**
 * @generated SignedSource<<7b1a6c1d304b13b267743f64b62e07d1>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type PackageKind = "OPA_POLICY" | "%future added value";
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
    readonly kind?: PackageKind;
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
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "kind",
  "storageKey": null
},
v6 = [
  {
    "kind": "Variable",
    "name": "id",
    "variableName": "versionId"
  }
],
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "version",
  "storageKey": null
},
v8 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "status",
  "storageKey": null
},
v9 = {
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
              (v4/*: any*/),
              (v5/*: any*/)
            ],
            "type": "Package",
            "abstractKey": null
          }
        ],
        "storageKey": null
      },
      {
        "alias": null,
        "args": (v6/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          {
            "kind": "InlineFragment",
            "selections": [
              (v2/*: any*/),
              (v7/*: any*/),
              (v8/*: any*/)
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
          (v9/*: any*/),
          (v2/*: any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              (v3/*: any*/),
              (v4/*: any*/),
              (v5/*: any*/)
            ],
            "type": "Package",
            "abstractKey": null
          }
        ],
        "storageKey": null
      },
      {
        "alias": null,
        "args": (v6/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          (v9/*: any*/),
          (v2/*: any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              (v7/*: any*/),
              (v8/*: any*/)
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
    "cacheID": "2bed401e8f199cd544d011a9b5248ee7",
    "id": null,
    "metadata": {},
    "name": "EditPackageVersionQuery",
    "operationKind": "query",
    "text": "query EditPackageVersionQuery(\n  $packageId: String!\n  $versionId: String!\n) {\n  pkg: node(id: $packageId) {\n    __typename\n    ... on Package {\n      id\n      name\n      groupPath\n      kind\n    }\n    id\n  }\n  node(id: $versionId) {\n    __typename\n    ... on PackageVersion {\n      id\n      version\n      status\n    }\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "dd2ca9847630de07c6040bceded4dc2a";

export default node;
