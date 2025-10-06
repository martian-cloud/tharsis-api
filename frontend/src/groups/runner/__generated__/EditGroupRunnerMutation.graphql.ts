/**
 * @generated SignedSource<<ce52b6911756c3bf7e634cc9ecf3331e>>
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
export type EditGroupRunnerMutation$variables = {
  input: UpdateRunnerInput;
};
export type EditGroupRunnerMutation$data = {
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
export type EditGroupRunnerMutation = {
  response: EditGroupRunnerMutation$data;
  variables: EditGroupRunnerMutation$variables;
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
    "name": "EditGroupRunnerMutation",
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
    "name": "EditGroupRunnerMutation",
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
    "cacheID": "7c93d7ef685ea763bbd625606db296f0",
    "id": null,
    "metadata": {},
    "name": "EditGroupRunnerMutation",
    "operationKind": "mutation",
    "text": "mutation EditGroupRunnerMutation(\n  $input: UpdateRunnerInput!\n) {\n  updateRunner(input: $input) {\n    runner {\n      ...RunnerListItemFragment_runner\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment RunnerListItemFragment_runner on Runner {\n  metadata {\n    createdAt\n  }\n  id\n  name\n  disabled\n  createdBy\n  groupPath\n}\n"
  }
};
})();

(node as any).hash = "cfa5a675f7f0962d6cdb99a288016c74";

export default node;
