/**
 * @generated SignedSource<<8d4ae647ee7509a73a08aadd894fdef6>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type JobLogStreamSubscriptionInput = {
  jobId: string;
  lastSeenLogSize?: number | null | undefined;
};
export type JobLogsSubscription$variables = {
  input: JobLogStreamSubscriptionInput;
};
export type JobLogsSubscription$data = {
  readonly jobLogStreamEvents: {
    readonly completed: boolean;
    readonly data: {
      readonly logs: string;
      readonly offset: number;
    } | null | undefined;
    readonly size: number;
  };
};
export type JobLogsSubscription = {
  response: JobLogsSubscription$data;
  variables: JobLogsSubscription$variables;
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
    "concreteType": "JobLogStreamEvent",
    "kind": "LinkedField",
    "name": "jobLogStreamEvents",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "size",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "completed",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "concreteType": "JobLogStreamEventData",
        "kind": "LinkedField",
        "name": "data",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "offset",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "logs",
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
    "name": "JobLogsSubscription",
    "selections": (v1/*: any*/),
    "type": "Subscription",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "JobLogsSubscription",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "5c53020c1b1272c7f93bf58ada9bc3d9",
    "id": null,
    "metadata": {},
    "name": "JobLogsSubscription",
    "operationKind": "subscription",
    "text": "subscription JobLogsSubscription(\n  $input: JobLogStreamSubscriptionInput!\n) {\n  jobLogStreamEvents(input: $input) {\n    size\n    completed\n    data {\n      offset\n      logs\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "30862c6d8ed3d6f81775d6fa28488862";

export default node;
