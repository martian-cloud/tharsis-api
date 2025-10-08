/**
 * @generated SignedSource<<ea3ac847270127ce497b3069c60ec374>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type WorkspaceNotificationPreferenceQuery$variables = {
  workspacePath: string;
};
export type WorkspaceNotificationPreferenceQuery$data = {
  readonly userPreferences: {
    readonly workspacePreferences: {
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
export type WorkspaceNotificationPreferenceQuery = {
  response: WorkspaceNotificationPreferenceQuery$data;
  variables: WorkspaceNotificationPreferenceQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "workspacePath"
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
    "variableName": "workspacePath"
  }
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "WorkspaceNotificationPreferenceQuery",
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
            "name": "workspacePreferences",
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
    "name": "WorkspaceNotificationPreferenceQuery",
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
            "name": "workspacePreferences",
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
    "cacheID": "a7e87f602f6c423d25d01b55ebca3b89",
    "id": null,
    "metadata": {},
    "name": "WorkspaceNotificationPreferenceQuery",
    "operationKind": "query",
    "text": "query WorkspaceNotificationPreferenceQuery(\n  $workspacePath: String!\n) {\n  userPreferences {\n    workspacePreferences(first: 1, path: $workspacePath) {\n      edges {\n        node {\n          notificationPreference {\n            ...NotificationButtonFragment_notificationPreference\n          }\n          id\n        }\n      }\n    }\n  }\n}\n\nfragment NotificationButtonFragment_notificationPreference on UserNotificationPreference {\n  scope\n  inherited\n  namespacePath\n  global\n  customEvents {\n    failedRun\n  }\n}\n"
  }
};
})();

(node as any).hash = "168afb4a28ec82fb030cf286a42309e2";

export default node;
