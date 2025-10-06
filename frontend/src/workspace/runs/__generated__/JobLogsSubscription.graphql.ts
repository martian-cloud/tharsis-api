/**
 * @generated SignedSource<<3e029dd7420e492e28cd5258aaca03c8>>
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
    "cacheID": "eb1d30d6cb57a80d14cac615936c9d2d",
    "id": null,
    "metadata": {},
    "name": "JobLogsSubscription",
    "operationKind": "subscription",
    "text": "subscription JobLogsSubscription(\n  $input: JobLogStreamSubscriptionInput!\n) {\n  jobLogStreamEvents(input: $input) {\n    size\n  }\n}\n"
  }
};
})();

(node as any).hash = "8dec75d5dcbc78eb86dd059a15ac0582";

export default node;
