/**
 * @generated SignedSource<<2a1ed2f73d2847ff5887215495a4300e>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type PolicyEnforcementLevel = "ADVISORY" | "HARD_MANDATORY" | "SOFT_MANDATORY" | "%future added value";
export type PolicyKind = "OPA" | "%future added value";
export type PolicyScopeRuleAction = "EXCLUDE" | "INCLUDE" | "%future added value";
export type PolicyScopeRuleType = "GROUP" | "MANAGED_IDENTITY" | "WORKSPACE" | "%future added value";
export type PolicyStage = "POST_APPLY" | "POST_PLAN" | "PRE_PLAN" | "%future added value";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type SpeculativeRunEnforcementLevel = "ADVISORY" | "HARD_MANDATORY" | "%future added value";
export type UpdatePolicyInput = {
  allowedServiceAccounts?: ReadonlyArray<string> | null | undefined;
  allowedTeams?: ReadonlyArray<string> | null | undefined;
  allowedUsers?: ReadonlyArray<string> | null | undefined;
  clientMutationId?: string | null | undefined;
  description?: string | null | undefined;
  id: string;
  opaData?: OPAPolicyDataInput | null | undefined;
  requiredApprovals?: number | null | undefined;
  scope?: ReadonlyArray<PolicyScopeRuleInput> | null | undefined;
};
export type OPAPolicyDataInput = {
  enforcementLevel: PolicyEnforcementLevel;
  packageDigest?: string | null | undefined;
  packageSource: string;
  packageVersionConstraint?: string | null | undefined;
  speculativeRunEnforcementLevel: SpeculativeRunEnforcementLevel;
  stage: PolicyStage;
};
export type PolicyScopeRuleInput = {
  action: PolicyScopeRuleAction;
  pattern: string;
  type: PolicyScopeRuleType;
};
export type EditPolicyMutation$variables = {
  input: UpdatePolicyInput;
};
export type EditPolicyMutation$data = {
  readonly updatePolicy: {
    readonly policy: {
      readonly allowedServiceAccounts: ReadonlyArray<{
        readonly id: string;
        readonly name: string;
        readonly resourcePath: string;
      }>;
      readonly allowedTeams: ReadonlyArray<{
        readonly id: string;
        readonly name: string;
      }>;
      readonly allowedUsers: ReadonlyArray<{
        readonly email: string;
        readonly id: string;
        readonly username: string;
      }>;
      readonly createdBy: string;
      readonly description: string;
      readonly groupPath: string;
      readonly id: string;
      readonly kind: PolicyKind;
      readonly name: string;
      readonly opaData: {
        readonly enforcementLevel: PolicyEnforcementLevel;
        readonly packageDigest: string | null | undefined;
        readonly packageSource: string;
        readonly packageVersionConstraint: string | null | undefined;
        readonly speculativeRunEnforcementLevel: SpeculativeRunEnforcementLevel;
        readonly stage: PolicyStage;
      } | null | undefined;
      readonly requiredApprovals: number;
      readonly scope: ReadonlyArray<{
        readonly action: PolicyScopeRuleAction;
        readonly pattern: string;
        readonly type: PolicyScopeRuleType;
      }>;
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type EditPolicyMutation = {
  response: EditPolicyMutation$data;
  variables: EditPolicyMutation$variables;
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
  "name": "name",
  "storageKey": null
},
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "type",
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
    "concreteType": "PolicyMutationPayload",
    "kind": "LinkedField",
    "name": "updatePolicy",
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
          (v2/*: any*/),
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
            "kind": "ScalarField",
            "name": "kind",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "concreteType": "OPAPolicyData",
            "kind": "LinkedField",
            "name": "opaData",
            "plural": false,
            "selections": [
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "packageSource",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "packageVersionConstraint",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "packageDigest",
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
                "name": "speculativeRunEnforcementLevel",
                "storageKey": null
              }
            ],
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
            "name": "requiredApprovals",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "concreteType": "PolicyScopeRule",
            "kind": "LinkedField",
            "name": "scope",
            "plural": true,
            "selections": [
              (v3/*: any*/),
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "action",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "pattern",
                "storageKey": null
              }
            ],
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
                "name": "email",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "username",
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
              (v2/*: any*/)
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
              (v2/*: any*/),
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "resourcePath",
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
          (v3/*: any*/)
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
    "name": "EditPolicyMutation",
    "selections": (v4/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "EditPolicyMutation",
    "selections": (v4/*: any*/)
  },
  "params": {
    "cacheID": "02be772cccee056b10090add91c0cf7c",
    "id": null,
    "metadata": {},
    "name": "EditPolicyMutation",
    "operationKind": "mutation",
    "text": "mutation EditPolicyMutation(\n  $input: UpdatePolicyInput!\n) {\n  updatePolicy(input: $input) {\n    policy {\n      id\n      name\n      description\n      kind\n      opaData {\n        packageSource\n        packageVersionConstraint\n        packageDigest\n        stage\n        enforcementLevel\n        speculativeRunEnforcementLevel\n      }\n      createdBy\n      requiredApprovals\n      scope {\n        type\n        action\n        pattern\n      }\n      groupPath\n      allowedUsers {\n        id\n        email\n        username\n      }\n      allowedTeams {\n        id\n        name\n      }\n      allowedServiceAccounts {\n        id\n        name\n        resourcePath\n      }\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "54bbf8e6b17c8caf999d461437a15b83";

export default node;
