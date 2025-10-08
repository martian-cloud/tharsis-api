/**
 * @generated SignedSource<<7b65155e2d41758777e864c86c26c763>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type RunnerType = "group" | "shared" | "%future added value";
export type RunnerSessionEventSubscriptionInput = {
  groupId?: string | null | undefined;
  runnerId?: string | null | undefined;
  runnerType?: RunnerType | null | undefined;
};
export type RunnerDetailsSessionEventsSubscription$variables = {
  input: RunnerSessionEventSubscriptionInput;
};
export type RunnerDetailsSessionEventsSubscription$data = {
  readonly runnerSessionEvents: {
    readonly action: string;
    readonly runnerSession: {
      readonly id: string;
      readonly " $fragmentSpreads": FragmentRefs<"RunnerSessionListItemFragment">;
    };
  };
};
export type RunnerDetailsSessionEventsSubscription = {
  response: RunnerDetailsSessionEventsSubscription$data;
  variables: RunnerDetailsSessionEventsSubscription$variables;
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
    "kind": "Variable",
    "name": "input",
    "variableName": "input"
  }
],
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "action",
  "storageKey": null
},
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "RunnerDetailsSessionEventsSubscription",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "RunnerSessionEvent",
        "kind": "LinkedField",
        "name": "runnerSessionEvents",
        "plural": false,
        "selections": [
          (v2/*: any*/),
          {
            "alias": null,
            "args": null,
            "concreteType": "RunnerSession",
            "kind": "LinkedField",
            "name": "runnerSession",
            "plural": false,
            "selections": [
              (v3/*: any*/),
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "RunnerSessionListItemFragment"
              }
            ],
            "storageKey": null
          }
        ],
        "storageKey": null
      }
    ],
    "type": "Subscription",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "RunnerDetailsSessionEventsSubscription",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "RunnerSessionEvent",
        "kind": "LinkedField",
        "name": "runnerSessionEvents",
        "plural": false,
        "selections": [
          (v2/*: any*/),
          {
            "alias": null,
            "args": null,
            "concreteType": "RunnerSession",
            "kind": "LinkedField",
            "name": "runnerSession",
            "plural": false,
            "selections": [
              (v3/*: any*/),
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "lastContacted",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "active",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "internal",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "errorCount",
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
                    "name": "updatedAt",
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
    "cacheID": "618d9867c5c22cb871c3175b50b85e4d",
    "id": null,
    "metadata": {},
    "name": "RunnerDetailsSessionEventsSubscription",
    "operationKind": "subscription",
    "text": "subscription RunnerDetailsSessionEventsSubscription(\n  $input: RunnerSessionEventSubscriptionInput!\n) {\n  runnerSessionEvents(input: $input) {\n    action\n    runnerSession {\n      id\n      ...RunnerSessionListItemFragment\n    }\n  }\n}\n\nfragment RunnerSessionListItemFragment on RunnerSession {\n  id\n  lastContacted\n  active\n  internal\n  errorCount\n  metadata {\n    updatedAt\n  }\n}\n"
  }
};
})();

(node as any).hash = "37579adfe558b149b29ae4d7e81a4ecd";

export default node;
