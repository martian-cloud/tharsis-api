/**
 * @generated SignedSource<<cd9a203fecf1347da11de77d0c0aa2b5>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type CancelAgentRunInput = {
  clientMutationId?: string | null | undefined;
  runId: string;
};
export type agentSessionCancelRunMutation$variables = {
  input: CancelAgentRunInput;
};
export type agentSessionCancelRunMutation$data = {
  readonly cancelAgentRun: {
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
export type agentSessionCancelRunMutation = {
  response: agentSessionCancelRunMutation$data;
  variables: agentSessionCancelRunMutation$variables;
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
    "concreteType": "CancelAgentRunPayload",
    "kind": "LinkedField",
    "name": "cancelAgentRun",
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
    "name": "agentSessionCancelRunMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "agentSessionCancelRunMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "ced5ca448b8dd25babf32c7864d91bcb",
    "id": null,
    "metadata": {},
    "name": "agentSessionCancelRunMutation",
    "operationKind": "mutation",
    "text": "mutation agentSessionCancelRunMutation(\n  $input: CancelAgentRunInput!\n) {\n  cancelAgentRun(input: $input) {\n    run {\n      id\n      status\n    }\n    problems {\n      message\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "c748728400ab0a0dbc77d7ce9a7f0e1e";

export default node;
