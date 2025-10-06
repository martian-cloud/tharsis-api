/**
 * @generated SignedSource<<de59aa86d7bdc711de7099313535a0dd>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type AssessWorkspaceInput = {
  clientMutationId?: string | null | undefined;
  workspaceId?: string | null | undefined;
  workspacePath?: string | null | undefined;
};
export type WorkspaceDetailsDriftDetectionMutation$variables = {
  input: AssessWorkspaceInput;
};
export type WorkspaceDetailsDriftDetectionMutation$data = {
  readonly assessWorkspace: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly run: {
      readonly id: string;
      readonly workspace: {
        readonly fullPath: string;
        readonly " $fragmentSpreads": FragmentRefs<"WorkspaceDetailsDriftDetectionFragment_workspace">;
      };
    } | null | undefined;
  };
};
export type WorkspaceDetailsDriftDetectionMutation = {
  response: WorkspaceDetailsDriftDetectionMutation$data;
  variables: WorkspaceDetailsDriftDetectionMutation$variables;
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
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "fullPath",
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
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "WorkspaceDetailsDriftDetectionMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "AssessWorkspacePayload",
        "kind": "LinkedField",
        "name": "assessWorkspace",
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
              (v2/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "Workspace",
                "kind": "LinkedField",
                "name": "workspace",
                "plural": false,
                "selections": [
                  (v3/*: any*/),
                  {
                    "args": null,
                    "kind": "FragmentSpread",
                    "name": "WorkspaceDetailsDriftDetectionFragment_workspace"
                  }
                ],
                "storageKey": null
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
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "WorkspaceDetailsDriftDetectionMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "AssessWorkspacePayload",
        "kind": "LinkedField",
        "name": "assessWorkspace",
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
              (v2/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "Workspace",
                "kind": "LinkedField",
                "name": "workspace",
                "plural": false,
                "selections": [
                  (v3/*: any*/),
                  (v2/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "WorkspaceAssessment",
                    "kind": "LinkedField",
                    "name": "assessment",
                    "plural": false,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "hasDrift",
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "startedAt",
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "completedAt",
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "Run",
                        "kind": "LinkedField",
                        "name": "run",
                        "plural": false,
                        "selections": [
                          (v2/*: any*/)
                        ],
                        "storageKey": null
                      },
                      (v2/*: any*/)
                    ],
                    "storageKey": null
                  }
                ],
                "storageKey": null
              }
            ],
            "storageKey": null
          },
          (v4/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "eaf213df747f7e0223d522487d67e984",
    "id": null,
    "metadata": {},
    "name": "WorkspaceDetailsDriftDetectionMutation",
    "operationKind": "mutation",
    "text": "mutation WorkspaceDetailsDriftDetectionMutation(\n  $input: AssessWorkspaceInput!\n) {\n  assessWorkspace(input: $input) {\n    run {\n      id\n      workspace {\n        fullPath\n        ...WorkspaceDetailsDriftDetectionFragment_workspace\n        id\n      }\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment WorkspaceDetailsDriftDetectionFragment_workspace on Workspace {\n  id\n  fullPath\n  assessment {\n    hasDrift\n    startedAt\n    completedAt\n    run {\n      id\n    }\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "d9e6d80dd5e9e29f2813e9b79e3d36de";

export default node;
