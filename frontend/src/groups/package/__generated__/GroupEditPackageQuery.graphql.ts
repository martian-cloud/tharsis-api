/**
 * @generated SignedSource<<f00c0b5292f81f669e26407832193d91>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type PackageKind = "OPA_POLICY" | "%future added value";
export type PackageVisibility = "GLOBAL" | "PRIVATE" | "ROOT_GROUP" | "%future added value";
export type GroupEditPackageQuery$variables = {
  id: string;
};
export type GroupEditPackageQuery$data = {
  readonly node: {
    readonly allowMutableVersions?: boolean;
    readonly description?: string;
    readonly id?: string;
    readonly kind?: PackageKind;
    readonly name?: string;
    readonly visibility?: PackageVisibility;
  } | null | undefined;
};
export type GroupEditPackageQuery = {
  response: GroupEditPackageQuery$data;
  variables: GroupEditPackageQuery$variables;
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
  "name": "description",
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
  "kind": "ScalarField",
  "name": "visibility",
  "storageKey": null
},
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "allowMutableVersions",
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "GroupEditPackageQuery",
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
              (v6/*: any*/),
              (v7/*: any*/)
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
    "name": "GroupEditPackageQuery",
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
              (v6/*: any*/),
              (v7/*: any*/)
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
    "cacheID": "2a20f86cd8a44c441ab8b7c30ca6cb6e",
    "id": null,
    "metadata": {},
    "name": "GroupEditPackageQuery",
    "operationKind": "query",
    "text": "query GroupEditPackageQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ... on Package {\n      id\n      name\n      description\n      kind\n      visibility\n      allowMutableVersions\n    }\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "01b061ef44ea029aba9d2df16fba3ffb";

export default node;
