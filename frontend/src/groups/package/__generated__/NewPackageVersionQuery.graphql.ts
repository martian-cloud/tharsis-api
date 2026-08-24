/**
 * @generated SignedSource<<3e9d8efe4ec7938488c40a3b4f2386a3>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type PackageKind = "OPA_POLICY" | "%future added value";
export type PackageVersionStatus = "ERRORED" | "PENDING" | "UPLOADED" | "UPLOAD_IN_PROGRESS" | "%future added value";
export type NewPackageVersionQuery$variables = {
  id: string;
};
export type NewPackageVersionQuery$data = {
  readonly node: {
    readonly groupPath?: string;
    readonly id?: string;
    readonly kind?: PackageKind;
    readonly latestVersion?: {
      readonly id: string;
      readonly status: PackageVersionStatus;
    } | null | undefined;
    readonly name?: string;
  } | null | undefined;
};
export type NewPackageVersionQuery = {
  response: NewPackageVersionQuery$data;
  variables: NewPackageVersionQuery$variables;
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
v6 = {
  "alias": null,
  "args": null,
  "concreteType": "PackageVersion",
  "kind": "LinkedField",
  "name": "latestVersion",
  "plural": false,
  "selections": [
    (v2/*: any*/),
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "status",
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
    "name": "NewPackageVersionQuery",
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
              (v6/*: any*/)
            ],
            "type": "Package",
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
    "name": "NewPackageVersionQuery",
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
              (v6/*: any*/)
            ],
            "type": "Package",
            "abstractKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "0d3e60de2052b2d65897ee54dc3516b1",
    "id": null,
    "metadata": {},
    "name": "NewPackageVersionQuery",
    "operationKind": "query",
    "text": "query NewPackageVersionQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ... on Package {\n      id\n      name\n      groupPath\n      kind\n      latestVersion {\n        id\n        status\n      }\n    }\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "7a42f4a8cfc5e7e5caf00396d559bc44";

export default node;
