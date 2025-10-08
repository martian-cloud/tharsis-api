/**
 * @generated SignedSource<<a23a51360dd2945a5e5be219fffa2fbd>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type JobType = "apply" | "plan" | "%future added value";
export type ManagedIdentityAccessRuleType = "eligible_principals" | "module_attestation" | "%future added value";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type UpdateManagedIdentityAccessRuleInput = {
  allowedServiceAccountIds?: ReadonlyArray<string> | null | undefined;
  allowedServiceAccounts?: ReadonlyArray<string> | null | undefined;
  allowedTeamIds?: ReadonlyArray<string> | null | undefined;
  allowedTeams?: ReadonlyArray<string> | null | undefined;
  allowedUserIds?: ReadonlyArray<string> | null | undefined;
  allowedUsers?: ReadonlyArray<string> | null | undefined;
  clientMutationId?: string | null | undefined;
  id: string;
  moduleAttestationPolicies?: ReadonlyArray<ManagedIdentityAccessRuleModuleAttestationPolicyInput> | null | undefined;
  runStage: JobType;
  verifyStateLineage?: boolean | null | undefined;
};
export type ManagedIdentityAccessRuleModuleAttestationPolicyInput = {
  predicateType?: string | null | undefined;
  publicKey: string;
};
export type ManagedIdentityRulesUpdateRuleMutation$variables = {
  input: UpdateManagedIdentityAccessRuleInput;
};
export type ManagedIdentityRulesUpdateRuleMutation$data = {
  readonly updateManagedIdentityAccessRule: {
    readonly accessRule: {
      readonly allowedServiceAccounts: ReadonlyArray<{
        readonly id: string;
        readonly name: string;
        readonly resourcePath: string;
      }> | null | undefined;
      readonly allowedTeams: ReadonlyArray<{
        readonly id: string;
        readonly name: string;
      }> | null | undefined;
      readonly allowedUsers: ReadonlyArray<{
        readonly email: string;
        readonly id: string;
        readonly username: string;
      }> | null | undefined;
      readonly id: string;
      readonly moduleAttestationPolicies: ReadonlyArray<{
        readonly predicateType: string | null | undefined;
        readonly publicKey: string;
      }> | null | undefined;
      readonly runStage: JobType;
      readonly type: ManagedIdentityAccessRuleType;
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type ManagedIdentityRulesUpdateRuleMutation = {
  response: ManagedIdentityRulesUpdateRuleMutation$data;
  variables: ManagedIdentityRulesUpdateRuleMutation$variables;
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
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "type",
  "storageKey": null
},
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "name",
  "storageKey": null
},
v4 = [
  {
    "alias": null,
    "args": [
      {
        "kind": "Variable",
        "name": "input",
        "variableName": "input"
      }
    ],
    "concreteType": "ManagedIdentityAccessRuleMutationPayload",
    "kind": "LinkedField",
    "name": "updateManagedIdentityAccessRule",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "ManagedIdentityAccessRule",
        "kind": "LinkedField",
        "name": "accessRule",
        "plural": false,
        "selections": [
          (v1/*: any*/),
          (v2/*: any*/),
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "runStage",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "concreteType": "User",
            "kind": "LinkedField",
            "name": "allowedUsers",
            "plural": true,
            "selections": [
              (v1/*: any*/),
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "username",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "email",
                "storageKey": null
              }
            ],
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "concreteType": "Team",
            "kind": "LinkedField",
            "name": "allowedTeams",
            "plural": true,
            "selections": [
              (v1/*: any*/),
              (v3/*: any*/)
            ],
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "concreteType": "ServiceAccount",
            "kind": "LinkedField",
            "name": "allowedServiceAccounts",
            "plural": true,
            "selections": [
              (v1/*: any*/),
              (v3/*: any*/),
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "resourcePath",
                "storageKey": null
              }
            ],
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "concreteType": "ManagedIdentityAccessRuleModuleAttestationPolicy",
            "kind": "LinkedField",
            "name": "moduleAttestationPolicies",
            "plural": true,
            "selections": [
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "publicKey",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "predicateType",
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
          (v2/*: any*/)
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
    "name": "ManagedIdentityRulesUpdateRuleMutation",
    "selections": (v4/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "ManagedIdentityRulesUpdateRuleMutation",
    "selections": (v4/*: any*/)
  },
  "params": {
    "cacheID": "928f4c58504e355940d191661277d772",
    "id": null,
    "metadata": {},
    "name": "ManagedIdentityRulesUpdateRuleMutation",
    "operationKind": "mutation",
    "text": "mutation ManagedIdentityRulesUpdateRuleMutation(\n  $input: UpdateManagedIdentityAccessRuleInput!\n) {\n  updateManagedIdentityAccessRule(input: $input) {\n    accessRule {\n      id\n      type\n      runStage\n      allowedUsers {\n        id\n        username\n        email\n      }\n      allowedTeams {\n        id\n        name\n      }\n      allowedServiceAccounts {\n        id\n        name\n        resourcePath\n      }\n      moduleAttestationPolicies {\n        publicKey\n        predicateType\n      }\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "984ff2d231523fa6559e828c30497f9e";

export default node;
