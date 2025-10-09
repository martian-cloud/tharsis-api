/**
 * @generated SignedSource<<3e8cf778b5447bbe6ca532b3550fa3fc>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type UserPreferencesQuery$variables = {
  after?: string | null | undefined;
  first?: number | null | undefined;
};
export type UserPreferencesQuery$data = {
  readonly " $fragmentSpreads": FragmentRefs<"UserPreferencesFragment_preferences">;
};
export type UserPreferencesQuery = {
  response: UserPreferencesQuery$data;
  variables: UserPreferencesQuery$variables;
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
  "name": "first"
},
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "__typename",
  "storageKey": null
},
v3 = [
  {
    "kind": "Variable",
    "name": "after",
    "variableName": "after"
  },
  {
    "kind": "Variable",
    "name": "first",
    "variableName": "first"
  },
  {
    "kind": "Literal",
    "name": "sort",
    "value": "CREATED_AT_DESC"
  }
],
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "UserPreferencesQuery",
    "selections": [
      {
        "args": null,
        "kind": "FragmentSpread",
        "name": "UserPreferencesFragment_preferences"
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [
      (v1/*: any*/),
      (v0/*: any*/)
    ],
    "kind": "Operation",
    "name": "UserPreferencesQuery",
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "UserPreferences",
        "kind": "LinkedField",
        "name": "userPreferences",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "GlobalUserPreferences",
            "kind": "LinkedField",
            "name": "globalPreferences",
            "plural": false,
            "selections": [
              {
                "alias": null,
                "args": null,
                "concreteType": "UserNotificationPreference",
                "kind": "LinkedField",
                "name": "notificationPreference",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "scope",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "inherited",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "namespacePath",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "global",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "UserNotificationPreferenceCustomEvents",
                    "kind": "LinkedField",
                    "name": "customEvents",
                    "plural": false,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "failedRun",
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
          }
        ],
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "concreteType": null,
        "kind": "LinkedField",
        "name": "me",
        "plural": false,
        "selections": [
          (v2/*: any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              {
                "alias": null,
                "args": (v3/*: any*/),
                "concreteType": "UserSessionConnection",
                "kind": "LinkedField",
                "name": "userSessions",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "UserSessionEdge",
                    "kind": "LinkedField",
                    "name": "edges",
                    "plural": true,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "UserSession",
                        "kind": "LinkedField",
                        "name": "node",
                        "plural": false,
                        "selections": [
                          (v4/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "userAgent",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "expiration",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "expired",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "current",
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
                          (v2/*: any*/)
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
                      }
                    ],
                    "storageKey": null
                  }
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": (v3/*: any*/),
                "filters": [
                  "sort"
                ],
                "handle": "connection",
                "key": "UserSessions_userSessions",
                "kind": "LinkedHandle",
                "name": "userSessions"
              },
              (v4/*: any*/)
            ],
            "type": "User",
            "abstractKey": null
          },
          {
            "kind": "InlineFragment",
            "selections": [
              (v4/*: any*/)
            ],
            "type": "Node",
            "abstractKey": "__isNode"
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "cd62b808705f13ae0e47159088188c43",
    "id": null,
    "metadata": {},
    "name": "UserPreferencesQuery",
    "operationKind": "query",
    "text": "query UserPreferencesQuery(\n  $first: Int\n  $after: String\n) {\n  ...UserPreferencesFragment_preferences\n}\n\nfragment GlobalNotificationPreferenceFragment_notificationPreference on GlobalUserPreferences {\n  notificationPreference {\n    ...NotificationButtonFragment_notificationPreference\n  }\n}\n\nfragment NotificationButtonFragment_notificationPreference on UserNotificationPreference {\n  scope\n  inherited\n  namespacePath\n  global\n  customEvents {\n    failedRun\n  }\n}\n\nfragment UserPreferencesFragment_preferences on Query {\n  userPreferences {\n    globalPreferences {\n      ...GlobalNotificationPreferenceFragment_notificationPreference\n    }\n  }\n  me {\n    __typename\n    ... on User {\n      ...UserSessionsFragment_user\n    }\n    ... on Node {\n      __isNode: __typename\n      id\n    }\n  }\n}\n\nfragment UserSessionFragment_session on UserSession {\n  id\n  userAgent\n  expiration\n  expired\n  current\n  metadata {\n    createdAt\n  }\n}\n\nfragment UserSessionsFragment_user on User {\n  userSessions(first: $first, after: $after, sort: CREATED_AT_DESC) {\n    edges {\n      node {\n        id\n        ...UserSessionFragment_session\n        __typename\n      }\n      cursor\n    }\n    pageInfo {\n      endCursor\n      hasNextPage\n    }\n  }\n  id\n}\n"
  }
};
})();

(node as any).hash = "13e3d622010a4f25d97aceb1a1dcb604";

export default node;
