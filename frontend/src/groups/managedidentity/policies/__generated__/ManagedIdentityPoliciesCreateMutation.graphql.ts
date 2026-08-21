/**
 * @generated SignedSource<<4fc75e6163f93797aae15a82b2096912>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type PolicyEnforcementLevel = "advisory" | "hard_mandatory" | "soft_mandatory" | "%future added value";
export type PolicyStage = "post_apply" | "post_plan" | "pre_plan" | "%future added value";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type CreatePolicyInput = {
  allowedServiceAccounts?: ReadonlyArray<string> | null | undefined;
  allowedTeams?: ReadonlyArray<string> | null | undefined;
  allowedUsers?: ReadonlyArray<string> | null | undefined;
  clientMutationId?: string | null | undefined;
  digest?: string | null | undefined;
  enforcementLevel: PolicyEnforcementLevel;
  groupId?: string | null | undefined;
  managedIdentityId?: string | null | undefined;
  packageId: string;
  requiredApprovals?: number | null | undefined;
  stage: PolicyStage;
  versionConstraint?: string | null | undefined;
  workspaceId?: string | null | undefined;
};
export type ManagedIdentityPoliciesCreateMutation$variables = {
  input: CreatePolicyInput;
};
export type ManagedIdentityPoliciesCreateMutation$data = {
  readonly createPolicy: {
    readonly policy: {
      readonly createdBy: string;
      readonly digest: string | null | undefined;
      readonly enforcementLevel: PolicyEnforcementLevel;
      readonly id: string;
      readonly package: {
        readonly id: string;
        readonly name: string;
      };
      readonly stage: PolicyStage;
      readonly versionConstraint: string | null | undefined;
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type ManagedIdentityPoliciesCreateMutation = {
  response: ManagedIdentityPoliciesCreateMutation$data;
  variables: ManagedIdentityPoliciesCreateMutation$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "input"
  }
],
v1 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v2 = [
  {
    "alias": null,
    "args": [
      {
        "kind": "Variable",
        "name": "input",
        "variableName": "input"
      }
    ],
    "concreteType": "PolicyMutationPayload",
    "kind": "LinkedField",
    "name": "createPolicy",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "Policy",
        "kind": "LinkedField",
        "name": "policy",
        "plural": false,
        "selections": [
          (v1/*: any*/),
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "versionConstraint",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "digest",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "stage",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "enforcementLevel",
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
            "concreteType": "Package",
            "kind": "LinkedField",
            "name": "package",
            "plural": false,
            "selections": [
              (v1/*: any*/),
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "name",
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
    "name": "ManagedIdentityPoliciesCreateMutation",
    "selections": (v2/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "ManagedIdentityPoliciesCreateMutation",
    "selections": (v2/*: any*/)
  },
  "params": {
    "cacheID": "229f11e27aaa8009d01c5d17e80dfa14",
    "id": null,
    "metadata": {},
    "name": "ManagedIdentityPoliciesCreateMutation",
    "operationKind": "mutation",
    "text": "mutation ManagedIdentityPoliciesCreateMutation(\n  $input: CreatePolicyInput!\n) {\n  createPolicy(input: $input) {\n    policy {\n      id\n      versionConstraint\n      digest\n      stage\n      enforcementLevel\n      createdBy\n      package {\n        id\n        name\n      }\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "a03deb63f80a0482715ea55d96ce4539";

export default node;
