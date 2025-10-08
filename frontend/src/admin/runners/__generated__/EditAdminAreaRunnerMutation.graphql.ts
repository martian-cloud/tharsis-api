/**
 * @generated SignedSource<<0944b31781e7b68b6f5eae6ad4e1624e>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type UpdateRunnerInput = {
  clientMutationId?: string | null | undefined;
  description: string;
  disabled?: boolean | null | undefined;
  id: string;
  metadata?: ResourceMetadataInput | null | undefined;
  runUntaggedJobs?: boolean | null | undefined;
  tags?: ReadonlyArray<string> | null | undefined;
};
export type ResourceMetadataInput = {
  version: string;
};
export type EditAdminAreaRunnerMutation$variables = {
  input: UpdateRunnerInput;
};
export type EditAdminAreaRunnerMutation$data = {
  readonly updateRunner: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly runner: {
      readonly " $fragmentSpreads": FragmentRefs<"RunnerListItemFragment_runner">;
    } | null | undefined;
  };
};
export type EditAdminAreaRunnerMutation = {
  response: EditAdminAreaRunnerMutation$data;
  variables: EditAdminAreaRunnerMutation$variables;
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
    "kind": "Variable",
    "name": "input",
    "variableName": "input"
  }
],
v2 = {
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
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "EditAdminAreaRunnerMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "UpdateRunnerPayload",
        "kind": "LinkedField",
        "name": "updateRunner",
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
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "RunnerListItemFragment_runner"
              }
            ],
            "storageKey": null
          },
          (v2/*: any*/)
        ],
        "storageKey": null
      }
    ],
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "EditAdminAreaRunnerMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "UpdateRunnerPayload",
        "kind": "LinkedField",
        "name": "updateRunner",
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
                    "name": "createdAt",
                    "storageKey": null
                  }
                ],
                "storageKey": null
              },
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
                "name": "name",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "disabled",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "createdBy",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "groupPath",
                "storageKey": null
              }
            ],
            "storageKey": null
          },
          (v2/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "cabe99bfbfc0959b15111dcb4e309da7",
    "id": null,
    "metadata": {},
    "name": "EditAdminAreaRunnerMutation",
    "operationKind": "mutation",
    "text": "mutation EditAdminAreaRunnerMutation(\n  $input: UpdateRunnerInput!\n) {\n  updateRunner(input: $input) {\n    runner {\n      ...RunnerListItemFragment_runner\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment RunnerListItemFragment_runner on Runner {\n  metadata {\n    createdAt\n  }\n  id\n  name\n  disabled\n  createdBy\n  groupPath\n}\n"
  }
};
})();

(node as any).hash = "29f47eed8e6a55e895e9ac0580aa70f1";

export default node;
