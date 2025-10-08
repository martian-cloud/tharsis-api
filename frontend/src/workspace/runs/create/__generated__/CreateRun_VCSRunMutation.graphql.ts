/**
 * @generated SignedSource<<41dd022f5a501095bc07dde588dba3cf>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type CreateVCSRunInput = {
  clientMutationId?: string | null | undefined;
  isDestroy?: boolean | null | undefined;
  referenceName?: string | null | undefined;
  workspaceId?: string | null | undefined;
  workspacePath?: string | null | undefined;
};
export type CreateRun_VCSRunMutation$variables = {
  input: CreateVCSRunInput;
};
export type CreateRun_VCSRunMutation$data = {
  readonly createVCSRun: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type CreateRun_VCSRunMutation = {
  response: CreateRun_VCSRunMutation$data;
  variables: CreateRun_VCSRunMutation$variables;
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
    "concreteType": "CreateVCSRunPayload",
    "kind": "LinkedField",
    "name": "createVCSRun",
    "plural": false,
    "selections": [
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
    "name": "CreateRun_VCSRunMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "CreateRun_VCSRunMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "b9a8e99974f3a96472bfc54c5e6c32be",
    "id": null,
    "metadata": {},
    "name": "CreateRun_VCSRunMutation",
    "operationKind": "mutation",
    "text": "mutation CreateRun_VCSRunMutation(\n  $input: CreateVCSRunInput!\n) {\n  createVCSRun(input: $input) {\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "982a69b1eedd59983a5ce8f206406c71";

export default node;
