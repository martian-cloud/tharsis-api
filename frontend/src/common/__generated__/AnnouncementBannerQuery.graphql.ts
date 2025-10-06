/**
 * @generated SignedSource<<d572559cb7999adf8d5bba352edd1a0d>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type AnnouncementType = "ERROR" | "INFO" | "SUCCESS" | "WARNING" | "%future added value";
export type AnnouncementBannerQuery$variables = Record<PropertyKey, never>;
export type AnnouncementBannerQuery$data = {
  readonly announcements: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly dismissible: boolean;
        readonly id: string;
        readonly message: string;
        readonly type: AnnouncementType;
      } | null | undefined;
    } | null | undefined> | null | undefined;
  };
};
export type AnnouncementBannerQuery = {
  response: AnnouncementBannerQuery$data;
  variables: AnnouncementBannerQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "alias": null,
    "args": [
      {
        "kind": "Literal",
        "name": "active",
        "value": true
      },
      {
        "kind": "Literal",
        "name": "first",
        "value": 5
      },
      {
        "kind": "Literal",
        "name": "sort",
        "value": "START_TIME_DESC"
      }
    ],
    "concreteType": "AnnouncementConnection",
    "kind": "LinkedField",
    "name": "announcements",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "AnnouncementEdge",
        "kind": "LinkedField",
        "name": "edges",
        "plural": true,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "Announcement",
            "kind": "LinkedField",
            "name": "node",
            "plural": false,
            "selections": [
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "id",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "message",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "dismissible",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "type",
                "storageKey": null
              }
            ],
            "storageKey": null
          }
        ],
        "storageKey": null
      }
    ],
    "storageKey": "announcements(active:true,first:5,sort:\"START_TIME_DESC\")"
  }
];
return {
  "fragment": {
    "argumentDefinitions": [],
    "kind": "Fragment",
    "metadata": null,
    "name": "AnnouncementBannerQuery",
    "selections": (v0/*: any*/),
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [],
    "kind": "Operation",
    "name": "AnnouncementBannerQuery",
    "selections": (v0/*: any*/)
  },
  "params": {
    "cacheID": "6e558ebe51928e9fc2773d7e2af12fbf",
    "id": null,
    "metadata": {},
    "name": "AnnouncementBannerQuery",
    "operationKind": "query",
    "text": "query AnnouncementBannerQuery {\n  announcements(active: true, sort: START_TIME_DESC, first: 5) {\n    edges {\n      node {\n        id\n        message\n        dismissible\n        type\n      }\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "e579ec09c6b3d5e5d95b198f5859208a";

export default node;
