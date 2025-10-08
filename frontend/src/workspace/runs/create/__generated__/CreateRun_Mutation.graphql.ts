/**
 * @generated SignedSource<<74d6a025d0a0c3c44a92df21d7b2aaee>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest, Mutation } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "%future added value";
export type VariableCategory = "environment" | "terraform" | "%future added value";
export type CreateRunInput = {
  clientMutationId?: string | null;
  comment?: string | null;
  configurationVersionId?: string | null;
  isDestroy?: boolean | null;
  moduleSource?: string | null;
  moduleVersion?: string | null;
  terraformVersion?: string | null;
  variables?: ReadonlyArray<RunVariableInput> | null;
  workspacePath: string;
};
export type RunVariableInput = {
  category: VariableCategory;
  hcl: boolean;
  key: string;
  value: string;
};
export type CreateRun_Mutation$variables = {
  input: CreateRunInput;
};
export type CreateRun_Mutation$data = {
  readonly createRun: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly run: {
      readonly id: string;
    } | null;
  };
};
export type CreateRun_Mutation = {
  response: CreateRun_Mutation$data;
  variables: CreateRun_Mutation$variables;
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
    "concreteType": "RunMutationPayload",
    "kind": "LinkedField",
    "name": "createRun",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "Run",
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
    "name": "CreateRun_Mutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "CreateRun_Mutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "51286b9299b1c671d29c7d3cd74b00d4",
    "id": null,
    "metadata": {},
    "name": "CreateRun_Mutation",
    "operationKind": "mutation",
    "text": "mutation CreateRun_Mutation(\n  $input: CreateRunInput!\n) {\n  createRun(input: $input) {\n    run {\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "20ec99d73f7acaa8593f93a42f1da96f";

export default node;
