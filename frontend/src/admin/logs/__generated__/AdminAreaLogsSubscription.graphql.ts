/**
 * @generated SignedSource<<f9a0667073dc3b8e47d9cf2c5ed45423>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type AdminLogTailLevel = "DEBUG" | "ERROR" | "INFO" | "WARN" | "%future added value";
export type AdminLogTailSubscriptionInput = {
  levels?: ReadonlyArray<AdminLogTailLevel> | null | undefined;
  search?: string | null | undefined;
};
export type AdminAreaLogsSubscription$variables = {
  input: AdminLogTailSubscriptionInput;
};
export type AdminAreaLogsSubscription$data = {
  readonly adminLogTailEvents: {
    readonly error: string | null | undefined;
    readonly logEntry: {
      readonly caller: string | null | undefined;
      readonly fields: string | null | undefined;
      readonly id: string;
      readonly level: AdminLogTailLevel;
      readonly message: string;
      readonly stack: string | null | undefined;
      readonly timestamp: any;
    } | null | undefined;
  };
};
export type AdminAreaLogsSubscription = {
  response: AdminAreaLogsSubscription$data;
  variables: AdminAreaLogsSubscription$variables;
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
    "concreteType": "AdminLogTailEvent",
    "kind": "LinkedField",
    "name": "adminLogTailEvents",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "AdminLogTailEntry",
        "kind": "LinkedField",
        "name": "logEntry",
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
            "name": "timestamp",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "level",
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
            "name": "caller",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "stack",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "fields",
            "storageKey": null
          }
        ],
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "error",
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
    "name": "AdminAreaLogsSubscription",
    "selections": (v1/*: any*/),
    "type": "Subscription",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "AdminAreaLogsSubscription",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "fd8c2ae7180a2dd25609881846013dc1",
    "id": null,
    "metadata": {},
    "name": "AdminAreaLogsSubscription",
    "operationKind": "subscription",
    "text": "subscription AdminAreaLogsSubscription(\n  $input: AdminLogTailSubscriptionInput!\n) {\n  adminLogTailEvents(input: $input) {\n    logEntry {\n      id\n      timestamp\n      level\n      message\n      caller\n      stack\n      fields\n    }\n    error\n  }\n}\n"
  }
};
})();

(node as any).hash = "d289ce5195127015eb8bb90db27acf5a";

export default node;
