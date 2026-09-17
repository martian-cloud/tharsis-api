/**
 * @generated SignedSource<<d7033e826bebd25e37708016fcf1374d>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type SetRunAutoApplyInput = {
  autoApply: boolean;
  clientMutationId?: string | null | undefined;
  runId: string;
};
export type RunDetailsSidebarSetRunAutoApplyMutation$variables = {
  input: SetRunAutoApplyInput;
};
export type RunDetailsSidebarSetRunAutoApplyMutation$data = {
  readonly setRunAutoApply: {
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
export type RunDetailsSidebarSetRunAutoApplyMutation = {
  response: RunDetailsSidebarSetRunAutoApplyMutation$data;
  variables: RunDetailsSidebarSetRunAutoApplyMutation$variables;
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
  "kind": "ScalarField",
  "name": "createdAt",
  "storageKey": null
},
v6 = [
  (v4/*: any*/),
  {
    "alias": null,
    "args": null,
    "concreteType": "ResourceMetadata",
    "kind": "LinkedField",
    "name": "metadata",
    "plural": false,
    "selections": [
      (v5/*: any*/)
    ],
    "storageKey": null
  },
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
],
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "stageName",
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "RunDetailsSidebarSetRunAutoApplyMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "RunMutationPayload",
        "kind": "LinkedField",
        "name": "setRunAutoApply",
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
    "name": "RunDetailsSidebarSetRunAutoApplyMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "RunMutationPayload",
        "kind": "LinkedField",
        "name": "setRunAutoApply",
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
                "name": "autoApply",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "hasAdvisoryFailures",
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
                "concreteType": "RunAnnotation",
                "kind": "LinkedField",
                "name": "annotations",
                "plural": true,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "key",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "value",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "link",
                    "storageKey": null
                  }
                ],
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
              {
                "alias": null,
                "args": null,
                "concreteType": "ResourceMetadata",
                "kind": "LinkedField",
                "name": "metadata",
                "plural": false,
                "selections": [
                  (v5/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "trn",
                    "storageKey": null
                  }
                ],
                "storageKey": null
              },
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
                "concreteType": "RunTaskStage",
                "kind": "LinkedField",
                "name": "taskStages",
                "plural": true,
                "selections": [
                  (v7/*: any*/),
                  (v4/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "PolicyCheck",
                    "kind": "LinkedField",
                    "name": "policyChecks",
                    "plural": true,
                    "selections": [
                      (v4/*: any*/),
                      (v7/*: any*/)
                    ],
                    "storageKey": null
                  }
                ],
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
    "cacheID": "5517361f7e36fdbde3a888d0856e119a",
    "id": null,
    "metadata": {},
    "name": "RunDetailsSidebarSetRunAutoApplyMutation",
    "operationKind": "mutation",
    "text": "mutation RunDetailsSidebarSetRunAutoApplyMutation(\n  $input: SetRunAutoApplyInput!\n) {\n  setRunAutoApply(input: $input) {\n    run {\n      ...RunDetailsSidebarFragment_details\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment RunAnnotationsFragment_run on Run {\n  annotations {\n    key\n    value\n    link\n  }\n}\n\nfragment RunDetailsSidebarFragment_details on Run {\n  id\n  status\n  createdBy\n  isDestroy\n  assessment\n  autoApply\n  hasAdvisoryFailures\n  moduleSource\n  moduleVersion\n  annotations {\n    key\n  }\n  ...RunAnnotationsFragment_run\n  workspace {\n    fullPath\n    id\n  }\n  metadata {\n    createdAt\n    trn\n  }\n  configurationVersion {\n    id\n  }\n  plan {\n    status\n    metadata {\n      createdAt\n    }\n    currentJob {\n      runnerPath\n      cancelRequested\n      id\n    }\n    id\n  }\n  taskStages {\n    stageName\n    status\n    policyChecks {\n      status\n      stageName\n    }\n  }\n  apply {\n    status\n    metadata {\n      createdAt\n    }\n    currentJob {\n      runnerPath\n      cancelRequested\n      id\n    }\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "4b6ec2058e314d97131063890a5621c2";

export default node;
