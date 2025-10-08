/**
 * @generated SignedSource<<ae7b360811d38008f55bf0bf32108386>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type UserNotificationPreferenceScope = "ALL" | "CUSTOM" | "NONE" | "PARTICIPATE" | "%future added value";
export type SetUserNotificationPreferenceInput = {
  clientMutationId?: string | null | undefined;
  customEvents?: UserNotificationPreferenceCustomEventsInput | null | undefined;
  inherit?: boolean | null | undefined;
  namespacePath?: string | null | undefined;
  scope?: UserNotificationPreferenceScope | null | undefined;
};
export type UserNotificationPreferenceCustomEventsInput = {
  failedRun: boolean;
};
export type NotificationPreferenceDialogMutation$variables = {
  input: SetUserNotificationPreferenceInput;
};
export type NotificationPreferenceDialogMutation$data = {
  readonly setUserNotificationPreference: {
    readonly preference: {
      readonly customEvents: {
        readonly failedRun: boolean;
      } | null | undefined;
      readonly global: boolean;
      readonly inherited: boolean;
      readonly namespacePath: string | null | undefined;
      readonly scope: UserNotificationPreferenceScope;
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type NotificationPreferenceDialogMutation = {
  response: NotificationPreferenceDialogMutation$data;
  variables: NotificationPreferenceDialogMutation$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "input"
  }
],
v1 = [
  {
    "alias": null,
    "args": [
      {
        "kind": "Variable",
        "name": "input",
        "variableName": "input"
      }
    ],
    "concreteType": "SetUserNotificationPreferencePayload",
    "kind": "LinkedField",
    "name": "setUserNotificationPreference",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "UserNotificationPreference",
        "kind": "LinkedField",
        "name": "preference",
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
            "name": "global",
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
        "concreteType": "Problem",
        "kind": "LinkedField",
        "name": "problems",
        "plural": true,
        "selections": [
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
            "name": "field",
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
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "NotificationPreferenceDialogMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "NotificationPreferenceDialogMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "0c5cecb73d50bcc264166994418c5f48",
    "id": null,
    "metadata": {},
    "name": "NotificationPreferenceDialogMutation",
    "operationKind": "mutation",
    "text": "mutation NotificationPreferenceDialogMutation(\n  $input: SetUserNotificationPreferenceInput!\n) {\n  setUserNotificationPreference(input: $input) {\n    preference {\n      scope\n      inherited\n      global\n      namespacePath\n      customEvents {\n        failedRun\n      }\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "0574297d05e96c995077bd3f27a2a4cf";

export default node;
