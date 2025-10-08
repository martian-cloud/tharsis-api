/**
 * @generated SignedSource<<03e8aa65db946fdb377cf23bedce87dc>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type CreateRunnerInput = {
  clientMutationId?: string | null | undefined;
  description: string;
  disabled?: boolean | null | undefined;
  groupId?: string | null | undefined;
  groupPath?: string | null | undefined;
  name: string;
  runUntaggedJobs: boolean;
  tags: ReadonlyArray<string>;
};
export type NewGroupRunnerMutation$variables = {
  connections: ReadonlyArray<string>;
  input: CreateRunnerInput;
};
export type NewGroupRunnerMutation$data = {
  readonly createRunner: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly runner: {
      readonly id: string;
      readonly " $fragmentSpreads": FragmentRefs<"RunnerListItemFragment_runner">;
    } | null | undefined;
  };
};
export type NewGroupRunnerMutation = {
  response: NewGroupRunnerMutation$data;
  variables: NewGroupRunnerMutation$variables;
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
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v4 = {
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
    "name": "NewGroupRunnerMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "CreateRunnerPayload",
        "kind": "LinkedField",
        "name": "createRunner",
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
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "RunnerListItemFragment_runner"
              }
            ],
            "storageKey": null
          },
          (v4/*: any*/)
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
    "name": "NewGroupRunnerMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "CreateRunnerPayload",
        "kind": "LinkedField",
        "name": "createRunner",
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
          {
            "alias": null,
            "args": null,
            "filters": null,
            "handle": "prependNode",
            "key": "",
            "kind": "LinkedHandle",
            "name": "runner",
            "handleArgs": [
              {
                "kind": "Variable",
                "name": "connections",
                "variableName": "connections"
              },
              {
                "kind": "Literal",
                "name": "edgeTypeName",
                "value": "RunnerEdge"
              }
            ]
          },
          (v4/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "1eb8a33035a95498d4975075debf0ade",
    "id": null,
    "metadata": {},
    "name": "NewGroupRunnerMutation",
    "operationKind": "mutation",
    "text": "mutation NewGroupRunnerMutation(\n  $input: CreateRunnerInput!\n) {\n  createRunner(input: $input) {\n    runner {\n      id\n      ...RunnerListItemFragment_runner\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment RunnerListItemFragment_runner on Runner {\n  metadata {\n    createdAt\n  }\n  id\n  name\n  disabled\n  createdBy\n  groupPath\n}\n"
  }
};
})();

(node as any).hash = "d89d18e52db540f0291643c9f451dad6";

export default node;
