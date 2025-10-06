/**
 * @generated SignedSource<<8d64d48bc4539de2e581967ffd99bf25>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupNotificationPreferenceQuery$variables = {
  groupPath: string;
};
export type GroupNotificationPreferenceQuery$data = {
  readonly userPreferences: {
    readonly groupPreferences: {
      readonly edges: ReadonlyArray<{
        readonly node: {
          readonly notificationPreference: {
            readonly " $fragmentSpreads": FragmentRefs<"NotificationButtonFragment_notificationPreference">;
          };
        };
      } | null | undefined> | null | undefined;
    };
  };
};
export type GroupNotificationPreferenceQuery = {
  response: GroupNotificationPreferenceQuery$data;
  variables: GroupNotificationPreferenceQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "groupPath"
  }
],
v1 = [
  {
    "kind": "Literal",
    "name": "first",
    "value": 1
  },
  {
    "kind": "Variable",
    "name": "path",
    "variableName": "groupPath"
  }
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "GroupNotificationPreferenceQuery",
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
            "args": (v1/*: any*/),
            "concreteType": "UserNamespacePreferenceConnection",
            "kind": "LinkedField",
            "name": "groupPreferences",
            "plural": false,
            "selections": [
              {
                "alias": null,
                "args": null,
                "concreteType": "UserNamespacePreferenceEdge",
                "kind": "LinkedField",
                "name": "edges",
                "plural": true,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "UserNamespacePreferences",
                    "kind": "LinkedField",
                    "name": "node",
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
                            "args": null,
                            "kind": "FragmentSpread",
                            "name": "NotificationButtonFragment_notificationPreference"
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
    "name": "GroupNotificationPreferenceQuery",
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
            "args": (v1/*: any*/),
            "concreteType": "UserNamespacePreferenceConnection",
            "kind": "LinkedField",
            "name": "groupPreferences",
            "plural": false,
            "selections": [
              {
                "alias": null,
                "args": null,
                "concreteType": "UserNamespacePreferenceEdge",
                "kind": "LinkedField",
                "name": "edges",
                "plural": true,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "UserNamespacePreferences",
                    "kind": "LinkedField",
                    "name": "node",
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
                      },
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "id",
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
      }
    ]
  },
  "params": {
    "cacheID": "bd485f063996facb75271772a79ef2f2",
    "id": null,
    "metadata": {},
    "name": "GroupNotificationPreferenceQuery",
    "operationKind": "query",
    "text": "query GroupNotificationPreferenceQuery(\n  $groupPath: String!\n) {\n  userPreferences {\n    groupPreferences(first: 1, path: $groupPath) {\n      edges {\n        node {\n          notificationPreference {\n            ...NotificationButtonFragment_notificationPreference\n          }\n          id\n        }\n      }\n    }\n  }\n}\n\nfragment NotificationButtonFragment_notificationPreference on UserNotificationPreference {\n  scope\n  inherited\n  namespacePath\n  global\n  customEvents {\n    failedRun\n  }\n}\n"
  }
};
})();

(node as any).hash = "bafbd2a66511c8f9b28e734ec768bae0";

export default node;
