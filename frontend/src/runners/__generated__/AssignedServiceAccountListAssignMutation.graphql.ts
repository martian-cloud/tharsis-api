/**
 * @generated SignedSource<<940c33ac33f094320df86ffd0d289852>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type AssignServiceAccountToRunnerInput = {
  clientMutationId?: string | null | undefined;
  runnerId?: string | null | undefined;
  runnerPath?: string | null | undefined;
  serviceAccountId?: string | null | undefined;
  serviceAccountPath?: string | null | undefined;
};
export type AssignedServiceAccountListAssignMutation$variables = {
  connections: ReadonlyArray<string>;
  input: AssignServiceAccountToRunnerInput;
};
export type AssignedServiceAccountListAssignMutation$data = {
  readonly assignServiceAccountToRunner: {
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
      readonly " $fragmentSpreads": FragmentRefs<"AssignedServiceAccountListItemFragment_assignedServiceAccount">;
    } | null | undefined;
  };
};
export type AssignedServiceAccountListAssignMutation = {
  response: AssignedServiceAccountListAssignMutation$data;
  variables: AssignedServiceAccountListAssignMutation$variables;
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
    "name": "AssignedServiceAccountListAssignMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "AssignServiceAccountToRunnerPayload",
        "kind": "LinkedField",
        "name": "assignServiceAccountToRunner",
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
              (v4/*: any*/),
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "AssignedServiceAccountListItemFragment_assignedServiceAccount"
              }
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
    "name": "AssignedServiceAccountListAssignMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "AssignServiceAccountToRunnerPayload",
        "kind": "LinkedField",
        "name": "assignServiceAccountToRunner",
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
                "kind": "ScalarField",
                "name": "name",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "resourcePath",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "groupPath",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "description",
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
            "filters": null,
            "handle": "prependNode",
            "key": "",
            "kind": "LinkedHandle",
            "name": "serviceAccount",
            "handleArgs": [
              {
                "kind": "Variable",
                "name": "connections",
                "variableName": "connections"
              },
              {
                "kind": "Literal",
                "name": "edgeTypeName",
                "value": "ServiceAccountEdge"
              }
            ]
          },
          (v5/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "4aeb5d8de0faf943697b5bbac4cd1176",
    "id": null,
    "metadata": {},
    "name": "AssignedServiceAccountListAssignMutation",
    "operationKind": "mutation",
    "text": "mutation AssignedServiceAccountListAssignMutation(\n  $input: AssignServiceAccountToRunnerInput!\n) {\n  assignServiceAccountToRunner(input: $input) {\n    runner {\n      assignedServiceAccounts(first: 0) {\n        totalCount\n      }\n      id\n    }\n    serviceAccount {\n      id\n      ...AssignedServiceAccountListItemFragment_assignedServiceAccount\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment AssignedServiceAccountListItemFragment_assignedServiceAccount on ServiceAccount {\n  id\n  name\n  resourcePath\n  groupPath\n  description\n  metadata {\n    updatedAt\n  }\n}\n"
  }
};
})();

(node as any).hash = "c865b1469678143e53e4e6f9f88ae181";

export default node;
