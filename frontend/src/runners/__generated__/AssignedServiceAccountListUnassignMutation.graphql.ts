/**
 * @generated SignedSource<<ab85daac4846e8d3d58a903296836494>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type AssignServiceAccountToRunnerInput = {
  clientMutationId?: string | null | undefined;
  runnerId?: string | null | undefined;
  runnerPath?: string | null | undefined;
  serviceAccountId?: string | null | undefined;
  serviceAccountPath?: string | null | undefined;
};
export type AssignedServiceAccountListUnassignMutation$variables = {
  connections: ReadonlyArray<string>;
  input: AssignServiceAccountToRunnerInput;
};
export type AssignedServiceAccountListUnassignMutation$data = {
  readonly unassignServiceAccountFromRunner: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly runner: {
      readonly assignedServiceAccounts: {
        readonly totalCount: number;
      };
    } | null | undefined;
    readonly serviceAccount: {
      readonly id: string;
    } | null | undefined;
  };
};
export type AssignedServiceAccountListUnassignMutation = {
  response: AssignedServiceAccountListUnassignMutation$data;
  variables: AssignedServiceAccountListUnassignMutation$variables;
};

const node: ConcreteRequest = (function(){
var v0 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "connections"
},
v1 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "input"
},
v2 = [
  {
    "kind": "Variable",
    "name": "input",
    "variableName": "input"
  }
],
v3 = {
  "alias": null,
  "args": [
    {
      "kind": "Literal",
      "name": "first",
      "value": 0
    }
  ],
  "concreteType": "ServiceAccountConnection",
  "kind": "LinkedField",
  "name": "assignedServiceAccounts",
  "plural": false,
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "totalCount",
      "storageKey": null
    }
  ],
  "storageKey": "assignedServiceAccounts(first:0)"
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v5 = {
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
};
return {
  "fragment": {
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "AssignedServiceAccountListUnassignMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "AssignServiceAccountToRunnerPayload",
        "kind": "LinkedField",
        "name": "unassignServiceAccountFromRunner",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "Runner",
            "kind": "LinkedField",
            "name": "runner",
            "plural": false,
            "selections": [
              (v3/*: any*/)
            ],
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "concreteType": "ServiceAccount",
            "kind": "LinkedField",
            "name": "serviceAccount",
            "plural": false,
            "selections": [
              (v4/*: any*/)
            ],
            "storageKey": null
          },
          (v5/*: any*/)
        ],
        "storageKey": null
      }
    ],
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [
      (v1/*: any*/),
      (v0/*: any*/)
    ],
    "kind": "Operation",
    "name": "AssignedServiceAccountListUnassignMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "AssignServiceAccountToRunnerPayload",
        "kind": "LinkedField",
        "name": "unassignServiceAccountFromRunner",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "Runner",
            "kind": "LinkedField",
            "name": "runner",
            "plural": false,
            "selections": [
              (v3/*: any*/),
              (v4/*: any*/)
            ],
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "concreteType": "ServiceAccount",
            "kind": "LinkedField",
            "name": "serviceAccount",
            "plural": false,
            "selections": [
              (v4/*: any*/),
              {
                "alias": null,
                "args": null,
                "filters": null,
                "handle": "deleteEdge",
                "key": "",
                "kind": "ScalarHandle",
                "name": "id",
                "handleArgs": [
                  {
                    "kind": "Variable",
                    "name": "connections",
                    "variableName": "connections"
                  }
                ]
              }
            ],
            "storageKey": null
          },
          (v5/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "210c470c05143d77e179f1b74d7353a9",
    "id": null,
    "metadata": {},
    "name": "AssignedServiceAccountListUnassignMutation",
    "operationKind": "mutation",
    "text": "mutation AssignedServiceAccountListUnassignMutation(\n  $input: AssignServiceAccountToRunnerInput!\n) {\n  unassignServiceAccountFromRunner(input: $input) {\n    runner {\n      assignedServiceAccounts(first: 0) {\n        totalCount\n      }\n      id\n    }\n    serviceAccount {\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "b0a415659db9743bcbb5dfb05384cb34";

export default node;
