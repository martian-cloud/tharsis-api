/**
 * @generated SignedSource<<08f3b17bd41ae8fdac083ca97b589201>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type CreateAgentRunInput = {
  clientMutationId?: string | null | undefined;
  context?: ReadonlyArray<string> | null | undefined;
  message: string;
  previousRunId?: string | null | undefined;
  sessionId: string;
};
export type agentSessionCreateRunMutation$variables = {
  input: CreateAgentRunInput;
};
export type agentSessionCreateRunMutation$data = {
  readonly createAgentRun: {
    readonly problems: ReadonlyArray<{
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly run: {
      readonly id: string;
      readonly status: string;
    } | null | undefined;
  };
};
export type agentSessionCreateRunMutation = {
  response: agentSessionCreateRunMutation$data;
  variables: agentSessionCreateRunMutation$variables;
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
    "concreteType": "CreateAgentRunPayload",
    "kind": "LinkedField",
    "name": "createAgentRun",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "AgentSessionRun",
        "kind": "LinkedField",
        "name": "run",
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
            "name": "status",
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
    "name": "agentSessionCreateRunMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "agentSessionCreateRunMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "d3befd6917ac8c874e72438169fe5cc6",
    "id": null,
    "metadata": {},
    "name": "agentSessionCreateRunMutation",
    "operationKind": "mutation",
    "text": "mutation agentSessionCreateRunMutation(\n  $input: CreateAgentRunInput!\n) {\n  createAgentRun(input: $input) {\n    run {\n      id\n      status\n    }\n    problems {\n      message\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "7aff5bc73de15b05ac4c764ecf193058";

export default node;
