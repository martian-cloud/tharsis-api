/**
 * @generated SignedSource<<c37ce6954ae6d34c1fd0a9e29fb43d23>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type CreateAgentSessionInput = {
  clientMutationId?: string | null | undefined;
};
export type agentSessionCreateSessionMutation$variables = {
  input: CreateAgentSessionInput;
};
export type agentSessionCreateSessionMutation$data = {
  readonly createAgentSession: {
    readonly problems: ReadonlyArray<{
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly session: {
      readonly id: string;
    } | null | undefined;
  };
};
export type agentSessionCreateSessionMutation = {
  response: agentSessionCreateSessionMutation$data;
  variables: agentSessionCreateSessionMutation$variables;
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
    "concreteType": "CreateAgentSessionPayload",
    "kind": "LinkedField",
    "name": "createAgentSession",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "AgentSession",
        "kind": "LinkedField",
        "name": "session",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "id",
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
    "name": "agentSessionCreateSessionMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "agentSessionCreateSessionMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "895c32661a170285745b71ffc8e9d22b",
    "id": null,
    "metadata": {},
    "name": "agentSessionCreateSessionMutation",
    "operationKind": "mutation",
    "text": "mutation agentSessionCreateSessionMutation(\n  $input: CreateAgentSessionInput!\n) {\n  createAgentSession(input: $input) {\n    session {\n      id\n    }\n    problems {\n      message\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "becd27bc5d8f9a8b2e0c2533e3018697";

export default node;
