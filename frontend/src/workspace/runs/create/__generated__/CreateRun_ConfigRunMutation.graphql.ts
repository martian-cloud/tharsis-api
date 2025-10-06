/**
 * @generated SignedSource<<2ee12f52767d48c1ff22763efcf49a5a>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type CreateConfigurationVersionInput = {
  clientMutationId?: string | null | undefined;
  speculative?: boolean | null | undefined;
  workspaceId?: string | null | undefined;
  workspacePath?: string | null | undefined;
};
export type CreateRun_ConfigRunMutation$variables = {
  input: CreateConfigurationVersionInput;
};
export type CreateRun_ConfigRunMutation$data = {
  readonly createConfigurationVersion: {
    readonly configurationVersion: {
      readonly id: string;
      readonly status: string;
      readonly vcsEvent: {
        readonly status: string;
        readonly type: string;
      } | null | undefined;
      readonly workspaceId: string;
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type CreateRun_ConfigRunMutation = {
  response: CreateRun_ConfigRunMutation$data;
  variables: CreateRun_ConfigRunMutation$variables;
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
  "name": "status",
  "storageKey": null
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "workspaceId",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "type",
  "storageKey": null
},
v6 = {
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
    (v5/*: any*/)
  ],
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "CreateRun_ConfigRunMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "ConfigurationVersionMutationPayload",
        "kind": "LinkedField",
        "name": "createConfigurationVersion",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "ConfigurationVersion",
            "kind": "LinkedField",
            "name": "configurationVersion",
            "plural": false,
            "selections": [
              (v2/*: any*/),
              (v3/*: any*/),
              (v4/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "VCSEvent",
                "kind": "LinkedField",
                "name": "vcsEvent",
                "plural": false,
                "selections": [
                  (v5/*: any*/),
                  (v3/*: any*/)
                ],
                "storageKey": null
              }
            ],
            "storageKey": null
          },
          (v6/*: any*/)
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
    "name": "CreateRun_ConfigRunMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "ConfigurationVersionMutationPayload",
        "kind": "LinkedField",
        "name": "createConfigurationVersion",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "ConfigurationVersion",
            "kind": "LinkedField",
            "name": "configurationVersion",
            "plural": false,
            "selections": [
              (v2/*: any*/),
              (v3/*: any*/),
              (v4/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "VCSEvent",
                "kind": "LinkedField",
                "name": "vcsEvent",
                "plural": false,
                "selections": [
                  (v5/*: any*/),
                  (v3/*: any*/),
                  (v2/*: any*/)
                ],
                "storageKey": null
              }
            ],
            "storageKey": null
          },
          (v6/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "cf22e8405dfed91f35326b27ff7f8275",
    "id": null,
    "metadata": {},
    "name": "CreateRun_ConfigRunMutation",
    "operationKind": "mutation",
    "text": "mutation CreateRun_ConfigRunMutation(\n  $input: CreateConfigurationVersionInput!\n) {\n  createConfigurationVersion(input: $input) {\n    configurationVersion {\n      id\n      status\n      workspaceId\n      vcsEvent {\n        type\n        status\n        id\n      }\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "e33302c53b0d275ec2d8f317c39d09df";

export default node;
