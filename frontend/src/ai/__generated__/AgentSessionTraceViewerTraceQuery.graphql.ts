/**
 * @generated SignedSource<<1e4e32cd77fc0e6ac0c67440c3517c8a>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type AgentTraceInput = {
  runId: string;
};
export type AgentSessionTraceViewerTraceQuery$variables = {
  input: AgentTraceInput;
};
export type AgentSessionTraceViewerTraceQuery$data = {
  readonly agentTrace: string | null | undefined;
};
export type AgentSessionTraceViewerTraceQuery = {
  response: AgentSessionTraceViewerTraceQuery$data;
  variables: AgentSessionTraceViewerTraceQuery$variables;
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
    "kind": "ScalarField",
    "name": "agentTrace",
    "storageKey": null
  }
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "AgentSessionTraceViewerTraceQuery",
    "selections": (v1/*: any*/),
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "AgentSessionTraceViewerTraceQuery",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "dc415b39799a04bd7598d8440b846d5e",
    "id": null,
    "metadata": {},
    "name": "AgentSessionTraceViewerTraceQuery",
    "operationKind": "query",
    "text": "query AgentSessionTraceViewerTraceQuery(\n  $input: AgentTraceInput!\n) {\n  agentTrace(input: $input)\n}\n"
  }
};
})();

(node as any).hash = "e38f323e547c764f64e51b8b2eafecf5";

export default node;
