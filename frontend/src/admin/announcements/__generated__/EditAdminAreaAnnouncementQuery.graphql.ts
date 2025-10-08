/**
 * @generated SignedSource<<0a2bdd384f753540c944cd3887484c33>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type AnnouncementType = "ERROR" | "INFO" | "SUCCESS" | "WARNING" | "%future added value";
export type EditAdminAreaAnnouncementQuery$variables = {
  id: string;
};
export type EditAdminAreaAnnouncementQuery$data = {
  readonly node: {
    readonly dismissible?: boolean;
    readonly endTime?: any | null | undefined;
    readonly id?: string;
    readonly message?: string;
    readonly startTime?: any;
    readonly type?: AnnouncementType;
  } | null | undefined;
};
export type EditAdminAreaAnnouncementQuery = {
  response: EditAdminAreaAnnouncementQuery$data;
  variables: EditAdminAreaAnnouncementQuery$variables;
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
  "name": "message",
  "storageKey": null
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "type",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "dismissible",
  "storageKey": null
},
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "startTime",
  "storageKey": null
},
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "endTime",
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "EditAdminAreaAnnouncementQuery",
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
            "type": "Announcement",
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
    "name": "EditAdminAreaAnnouncementQuery",
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
            "type": "Announcement",
            "abstractKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "5c24c8d6e501ef53fdce5b956de0051f",
    "id": null,
    "metadata": {},
    "name": "EditAdminAreaAnnouncementQuery",
    "operationKind": "query",
    "text": "query EditAdminAreaAnnouncementQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ... on Announcement {\n      id\n      message\n      type\n      dismissible\n      startTime\n      endTime\n    }\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "320b2709815f7ffbf8f077d1d73bd286";

export default node;
