/**
 * @generated SignedSource<<c9f0375cff7a002a766b6cb5599c0b47>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type CancelRunInput = {
  clientMutationId?: string | null | undefined;
  comment?: string | null | undefined;
  force?: boolean | null | undefined;
  runId: string;
};
export type RunDetailsSidebarCancelRunMutation$variables = {
  input: CancelRunInput;
};
export type RunDetailsSidebarCancelRunMutation$data = {
  readonly cancelRun: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly run: {
      readonly " $fragmentSpreads": FragmentRefs<"RunDetailsSidebarFragment_details">;
    } | null | undefined;
  };
};
export type RunDetailsSidebarCancelRunMutation = {
  response: RunDetailsSidebarCancelRunMutation$data;
  variables: RunDetailsSidebarCancelRunMutation$variables;
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
},
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
  "kind": "ScalarField",
  "name": "status",
  "storageKey": null
},
v5 = {
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
v6 = [
  (v4/*: any*/),
  (v5/*: any*/),
  {
    "alias": null,
    "args": null,
    "concreteType": "Job",
    "kind": "LinkedField",
    "name": "currentJob",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "runnerPath",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "cancelRequested",
        "storageKey": null
      },
      (v3/*: any*/)
    ],
    "storageKey": null
  },
  (v3/*: any*/)
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "RunDetailsSidebarCancelRunMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "RunMutationPayload",
        "kind": "LinkedField",
        "name": "cancelRun",
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
                "args": null,
                "kind": "FragmentSpread",
                "name": "RunDetailsSidebarFragment_details"
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
    "name": "RunDetailsSidebarCancelRunMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "RunMutationPayload",
        "kind": "LinkedField",
        "name": "cancelRun",
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
              (v3/*: any*/),
              (v4/*: any*/),
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
                "name": "isDestroy",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "assessment",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "moduleSource",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "moduleVersion",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "Workspace",
                "kind": "LinkedField",
                "name": "workspace",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "fullPath",
                    "storageKey": null
                  },
                  (v3/*: any*/)
                ],
                "storageKey": null
              },
              (v5/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "ConfigurationVersion",
                "kind": "LinkedField",
                "name": "configurationVersion",
                "plural": false,
                "selections": [
                  (v3/*: any*/)
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "Plan",
                "kind": "LinkedField",
                "name": "plan",
                "plural": false,
                "selections": (v6/*: any*/),
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "Apply",
                "kind": "LinkedField",
                "name": "apply",
                "plural": false,
                "selections": (v6/*: any*/),
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
    "cacheID": "0f357261c6f257eff4dfd76743ef4eff",
    "id": null,
    "metadata": {},
    "name": "RunDetailsSidebarCancelRunMutation",
    "operationKind": "mutation",
    "text": "mutation RunDetailsSidebarCancelRunMutation(\n  $input: CancelRunInput!\n) {\n  cancelRun(input: $input) {\n    run {\n      ...RunDetailsSidebarFragment_details\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment RunDetailsSidebarFragment_details on Run {\n  id\n  status\n  createdBy\n  isDestroy\n  assessment\n  moduleSource\n  moduleVersion\n  workspace {\n    fullPath\n    id\n  }\n  metadata {\n    createdAt\n  }\n  configurationVersion {\n    id\n  }\n  plan {\n    status\n    metadata {\n      createdAt\n    }\n    currentJob {\n      runnerPath\n      cancelRequested\n      id\n    }\n    id\n  }\n  apply {\n    status\n    metadata {\n      createdAt\n    }\n    currentJob {\n      runnerPath\n      cancelRequested\n      id\n    }\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "9c0a6e4965203a917e6eda1b0baa2945";

export default node;
