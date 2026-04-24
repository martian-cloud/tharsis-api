/**
 * @generated SignedSource<<1bf3764623adc706864d96ec6ddf0ce8>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type AgentSessionEventSubscriptionInput = {
  sessionId: string;
};
export type agentSessionEventsSubscription$variables = {
  input: AgentSessionEventSubscriptionInput;
};
export type agentSessionEventsSubscription$data = {
  readonly agentSessionEvents: {
    readonly __typename: "AgentSessionCustomEvent";
    readonly name: string;
    readonly type: string;
    readonly value: string | null | undefined;
  } | {
    readonly __typename: "AgentSessionReasoningEndEvent";
    readonly messageId: string;
    readonly type: string;
  } | {
    readonly __typename: "AgentSessionReasoningMessageContentEvent";
    readonly delta: string;
    readonly messageId: string;
    readonly type: string;
  } | {
    readonly __typename: "AgentSessionReasoningMessageEndEvent";
    readonly messageId: string;
    readonly type: string;
  } | {
    readonly __typename: "AgentSessionReasoningMessageStartEvent";
    readonly messageId: string;
    readonly role: string;
    readonly type: string;
  } | {
    readonly __typename: "AgentSessionReasoningStartEvent";
    readonly messageId: string;
    readonly type: string;
  } | {
    readonly __typename: "AgentSessionRunErrorEvent";
    readonly message: string;
    readonly runId: string;
    readonly threadId: string;
    readonly type: string;
  } | {
    readonly __typename: "AgentSessionRunFinishedEvent";
    readonly runId: string;
    readonly threadId: string;
    readonly type: string;
  } | {
    readonly __typename: "AgentSessionRunStartedEvent";
    readonly runId: string;
    readonly threadId: string;
    readonly type: string;
  } | {
    readonly __typename: "AgentSessionStepFinishedEvent";
    readonly stepName: string;
    readonly type: string;
  } | {
    readonly __typename: "AgentSessionStepStartedEvent";
    readonly stepName: string;
    readonly type: string;
  } | {
    readonly __typename: "AgentSessionTextMessageContentEvent";
    readonly delta: string;
    readonly messageId: string;
    readonly type: string;
  } | {
    readonly __typename: "AgentSessionTextMessageEndEvent";
    readonly messageId: string;
    readonly type: string;
  } | {
    readonly __typename: "AgentSessionTextMessageStartEvent";
    readonly messageId: string;
    readonly role: string;
    readonly type: string;
  } | {
    readonly __typename: "AgentSessionToolCallArgsEvent";
    readonly delta: string;
    readonly toolCallId: string;
    readonly type: string;
  } | {
    readonly __typename: "AgentSessionToolCallEndEvent";
    readonly toolCallId: string;
    readonly type: string;
  } | {
    readonly __typename: "AgentSessionToolCallResultEvent";
    readonly content: string;
    readonly messageId: string;
    readonly role: string;
    readonly toolCallId: string;
    readonly type: string;
  } | {
    readonly __typename: "AgentSessionToolCallStartEvent";
    readonly parentMessageId: string | null | undefined;
    readonly toolCallId: string;
    readonly toolCallName: string;
    readonly type: string;
  } | {
    // This will never be '%other', but we need some
    // value in case none of the concrete values match.
    readonly __typename: "%other";
  };
};
export type agentSessionEventsSubscription = {
  response: agentSessionEventsSubscription$data;
  variables: agentSessionEventsSubscription$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "input"
  }
],
v1 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "type",
  "storageKey": null
},
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "threadId",
  "storageKey": null
},
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "runId",
  "storageKey": null
},
v4 = [
  (v1/*: any*/),
  (v2/*: any*/),
  (v3/*: any*/)
],
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "messageId",
  "storageKey": null
},
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "role",
  "storageKey": null
},
v7 = [
  (v1/*: any*/),
  (v5/*: any*/),
  (v6/*: any*/)
],
v8 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "delta",
  "storageKey": null
},
v9 = [
  (v1/*: any*/),
  (v5/*: any*/),
  (v8/*: any*/)
],
v10 = [
  (v1/*: any*/),
  (v5/*: any*/)
],
v11 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "toolCallId",
  "storageKey": null
},
v12 = [
  (v1/*: any*/),
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "stepName",
    "storageKey": null
  }
],
v13 = [
  {
    "alias": null,
    "args": [
      {
        "kind": "Variable",
        "name": "input",
        "variableName": "input"
      }
    ],
    "concreteType": null,
    "kind": "LinkedField",
    "name": "agentSessionEvents",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "__typename",
        "storageKey": null
      },
      {
        "kind": "InlineFragment",
        "selections": (v4/*: any*/),
        "type": "AgentSessionRunStartedEvent",
        "abstractKey": null
      },
      {
        "kind": "InlineFragment",
        "selections": (v4/*: any*/),
        "type": "AgentSessionRunFinishedEvent",
        "abstractKey": null
      },
      {
        "kind": "InlineFragment",
        "selections": [
          (v1/*: any*/),
          (v2/*: any*/),
          (v3/*: any*/),
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "message",
            "storageKey": null
          }
        ],
        "type": "AgentSessionRunErrorEvent",
        "abstractKey": null
      },
      {
        "kind": "InlineFragment",
        "selections": [
          (v1/*: any*/),
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "name",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "value",
            "storageKey": null
          }
        ],
        "type": "AgentSessionCustomEvent",
        "abstractKey": null
      },
      {
        "kind": "InlineFragment",
        "selections": (v7/*: any*/),
        "type": "AgentSessionTextMessageStartEvent",
        "abstractKey": null
      },
      {
        "kind": "InlineFragment",
        "selections": (v9/*: any*/),
        "type": "AgentSessionTextMessageContentEvent",
        "abstractKey": null
      },
      {
        "kind": "InlineFragment",
        "selections": (v10/*: any*/),
        "type": "AgentSessionTextMessageEndEvent",
        "abstractKey": null
      },
      {
        "kind": "InlineFragment",
        "selections": [
          (v1/*: any*/),
          (v11/*: any*/),
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "toolCallName",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "parentMessageId",
            "storageKey": null
          }
        ],
        "type": "AgentSessionToolCallStartEvent",
        "abstractKey": null
      },
      {
        "kind": "InlineFragment",
        "selections": [
          (v1/*: any*/),
          (v11/*: any*/),
          (v8/*: any*/)
        ],
        "type": "AgentSessionToolCallArgsEvent",
        "abstractKey": null
      },
      {
        "kind": "InlineFragment",
        "selections": [
          (v1/*: any*/),
          (v11/*: any*/)
        ],
        "type": "AgentSessionToolCallEndEvent",
        "abstractKey": null
      },
      {
        "kind": "InlineFragment",
        "selections": [
          (v1/*: any*/),
          (v5/*: any*/),
          (v11/*: any*/),
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "content",
            "storageKey": null
          },
          (v6/*: any*/)
        ],
        "type": "AgentSessionToolCallResultEvent",
        "abstractKey": null
      },
      {
        "kind": "InlineFragment",
        "selections": (v10/*: any*/),
        "type": "AgentSessionReasoningStartEvent",
        "abstractKey": null
      },
      {
        "kind": "InlineFragment",
        "selections": (v7/*: any*/),
        "type": "AgentSessionReasoningMessageStartEvent",
        "abstractKey": null
      },
      {
        "kind": "InlineFragment",
        "selections": (v9/*: any*/),
        "type": "AgentSessionReasoningMessageContentEvent",
        "abstractKey": null
      },
      {
        "kind": "InlineFragment",
        "selections": (v10/*: any*/),
        "type": "AgentSessionReasoningMessageEndEvent",
        "abstractKey": null
      },
      {
        "kind": "InlineFragment",
        "selections": (v10/*: any*/),
        "type": "AgentSessionReasoningEndEvent",
        "abstractKey": null
      },
      {
        "kind": "InlineFragment",
        "selections": (v12/*: any*/),
        "type": "AgentSessionStepStartedEvent",
        "abstractKey": null
      },
      {
        "kind": "InlineFragment",
        "selections": (v12/*: any*/),
        "type": "AgentSessionStepFinishedEvent",
        "abstractKey": null
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
    "name": "agentSessionEventsSubscription",
    "selections": (v13/*: any*/),
    "type": "Subscription",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "agentSessionEventsSubscription",
    "selections": (v13/*: any*/)
  },
  "params": {
    "cacheID": "7472d376efae21032db1fc8f05b8894e",
    "id": null,
    "metadata": {},
    "name": "agentSessionEventsSubscription",
    "operationKind": "subscription",
    "text": "subscription agentSessionEventsSubscription(\n  $input: AgentSessionEventSubscriptionInput!\n) {\n  agentSessionEvents(input: $input) {\n    __typename\n    ... on AgentSessionRunStartedEvent {\n      type\n      threadId\n      runId\n    }\n    ... on AgentSessionRunFinishedEvent {\n      type\n      threadId\n      runId\n    }\n    ... on AgentSessionRunErrorEvent {\n      type\n      threadId\n      runId\n      message\n    }\n    ... on AgentSessionCustomEvent {\n      type\n      name\n      value\n    }\n    ... on AgentSessionTextMessageStartEvent {\n      type\n      messageId\n      role\n    }\n    ... on AgentSessionTextMessageContentEvent {\n      type\n      messageId\n      delta\n    }\n    ... on AgentSessionTextMessageEndEvent {\n      type\n      messageId\n    }\n    ... on AgentSessionToolCallStartEvent {\n      type\n      toolCallId\n      toolCallName\n      parentMessageId\n    }\n    ... on AgentSessionToolCallArgsEvent {\n      type\n      toolCallId\n      delta\n    }\n    ... on AgentSessionToolCallEndEvent {\n      type\n      toolCallId\n    }\n    ... on AgentSessionToolCallResultEvent {\n      type\n      messageId\n      toolCallId\n      content\n      role\n    }\n    ... on AgentSessionReasoningStartEvent {\n      type\n      messageId\n    }\n    ... on AgentSessionReasoningMessageStartEvent {\n      type\n      messageId\n      role\n    }\n    ... on AgentSessionReasoningMessageContentEvent {\n      type\n      messageId\n      delta\n    }\n    ... on AgentSessionReasoningMessageEndEvent {\n      type\n      messageId\n    }\n    ... on AgentSessionReasoningEndEvent {\n      type\n      messageId\n    }\n    ... on AgentSessionStepStartedEvent {\n      type\n      stepName\n    }\n    ... on AgentSessionStepFinishedEvent {\n      type\n      stepName\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "862bf3ae1580050d14bbfdbe35a537b6";

export default node;
