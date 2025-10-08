/**
 * @generated SignedSource<<bc5683da6015a31b7149d7bb9770263b>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type CancelRunInput = {
  clientMutationId?: string | null | undefined;
  comment?: string | null | undefined;
  force?: boolean | null | undefined;
  runId: string;
};
export type ForceCancelRunButtonCancelRunMutation$variables = {
  input: CancelRunInput;
};
export type ForceCancelRunButtonCancelRunMutation$data = {
  readonly cancelRun: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type ForceCancelRunButtonCancelRunMutation = {
  response: ForceCancelRunButtonCancelRunMutation$data;
  variables: ForceCancelRunButtonCancelRunMutation$variables;
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
    "name": "cancelRun",
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
    "name": "ForceCancelRunButtonCancelRunMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "ForceCancelRunButtonCancelRunMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "fc7284cb22eab2a8767c25c78f4f3f6d",
    "id": null,
    "metadata": {},
    "name": "ForceCancelRunButtonCancelRunMutation",
    "operationKind": "mutation",
    "text": "mutation ForceCancelRunButtonCancelRunMutation(\n  $input: CancelRunInput!\n) {\n  cancelRun(input: $input) {\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "19605a42839e08d302d9f0ac5e94c1e8";

export default node;
