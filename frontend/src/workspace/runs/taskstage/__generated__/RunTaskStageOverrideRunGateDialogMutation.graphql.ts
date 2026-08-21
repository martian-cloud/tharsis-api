/**
 * @generated SignedSource<<a6d41e05fb9d65b15251db894a5b6e65>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type RunGateStatus = "APPROVED" | "CANCELED" | "OVERRIDDEN" | "PENDING" | "%future added value";
export type OverrideRunGateInput = {
  clientMutationId?: string | null | undefined;
  comment?: string | null | undefined;
  gateId: string;
};
export type RunTaskStageOverrideRunGateDialogMutation$variables = {
  input: OverrideRunGateInput;
};
export type RunTaskStageOverrideRunGateDialogMutation$data = {
  readonly overrideRunGate: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly runGate: {
      readonly id: string;
      readonly metadata: {
        readonly updatedAt: any;
      };
      readonly overriddenBy: string | null | undefined;
      readonly overrideComment: string;
      readonly status: RunGateStatus;
    } | null | undefined;
  };
};
export type RunTaskStageOverrideRunGateDialogMutation = {
  response: RunTaskStageOverrideRunGateDialogMutation$data;
  variables: RunTaskStageOverrideRunGateDialogMutation$variables;
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
    "concreteType": "RunGateMutationPayload",
    "kind": "LinkedField",
    "name": "overrideRunGate",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "RunGate",
        "kind": "LinkedField",
        "name": "runGate",
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
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "overriddenBy",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "overrideComment",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "concreteType": "ResourceMetadata",
            "kind": "LinkedField",
            "name": "metadata",
            "plural": false,
            "selections": [
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "updatedAt",
                "storageKey": null
              }
            ],
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
    "name": "RunTaskStageOverrideRunGateDialogMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "RunTaskStageOverrideRunGateDialogMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "53ca47a6564bd48b286d8bbb8bad20a7",
    "id": null,
    "metadata": {},
    "name": "RunTaskStageOverrideRunGateDialogMutation",
    "operationKind": "mutation",
    "text": "mutation RunTaskStageOverrideRunGateDialogMutation(\n  $input: OverrideRunGateInput!\n) {\n  overrideRunGate(input: $input) {\n    runGate {\n      id\n      status\n      overriddenBy\n      overrideComment\n      metadata {\n        updatedAt\n      }\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "15397d3ede056d013d3ad8aa648bd642";

export default node;
