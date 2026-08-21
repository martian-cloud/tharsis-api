/**
 * @generated SignedSource<<acad2bb1cb9c278876be70f52bed1200>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type PolicyEnforcementLevel = "ADVISORY" | "HARD_MANDATORY" | "SOFT_MANDATORY" | "%future added value";
export type PolicyKind = "OPA" | "%future added value";
export type PolicyScopeRuleAction = "EXCLUDE" | "INCLUDE" | "%future added value";
export type PolicyScopeRuleType = "GROUP" | "MANAGED_IDENTITY" | "WORKSPACE" | "%future added value";
export type PolicyStage = "POST_APPLY" | "POST_PLAN" | "PRE_PLAN" | "%future added value";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type SpeculativeRunEnforcementLevel = "ADVISORY" | "HARD_MANDATORY" | "%future added value";
export type CreatePolicyInput = {
  allowedServiceAccounts?: ReadonlyArray<string> | null | undefined;
  allowedTeams?: ReadonlyArray<string> | null | undefined;
  allowedUsers?: ReadonlyArray<string> | null | undefined;
  clientMutationId?: string | null | undefined;
  description?: string | null | undefined;
  groupId: string;
  kind: PolicyKind;
  name: string;
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
export type NewPolicyCreateMutation$variables = {
  connections: ReadonlyArray<string>;
  input: CreatePolicyInput;
};
export type NewPolicyCreateMutation$data = {
  readonly createPolicy: {
    readonly policy: {
      readonly groupPath: string;
      readonly id: string;
      readonly " $fragmentSpreads": FragmentRefs<"PolicyDetailsFragment_policy">;
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type NewPolicyCreateMutation = {
  response: NewPolicyCreateMutation$data;
  variables: NewPolicyCreateMutation$variables;
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
  "kind": "ScalarField",
  "name": "groupPath",
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
},
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "name",
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
    "name": "NewPolicyCreateMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
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
              (v3/*: any*/),
              (v4/*: any*/),
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "PolicyDetailsFragment_policy"
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
    "argumentDefinitions": [
      (v1/*: any*/),
      (v0/*: any*/)
    ],
    "kind": "Operation",
    "name": "NewPolicyCreateMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
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
              (v3/*: any*/),
              (v4/*: any*/),
              (v7/*: any*/),
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
                "concreteType": "PolicyScopeRule",
                "kind": "LinkedField",
                "name": "scope",
                "plural": true,
                "selections": [
                  (v5/*: any*/),
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
                "concreteType": "User",
                "kind": "LinkedField",
                "name": "allowedUsers",
                "plural": true,
                "selections": [
                  (v3/*: any*/),
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
                  (v3/*: any*/),
                  (v7/*: any*/)
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
                  (v3/*: any*/),
                  (v7/*: any*/),
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
            "filters": null,
            "handle": "prependNode",
            "key": "",
            "kind": "LinkedHandle",
            "name": "policy",
            "handleArgs": [
              {
                "kind": "Variable",
                "name": "connections",
                "variableName": "connections"
              },
              {
                "kind": "Literal",
                "name": "edgeTypeName",
                "value": "PolicyEdge"
              }
            ]
          },
          (v6/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "9b5d5b1f46b8dab28c9cd8d5f11cf924",
    "id": null,
    "metadata": {},
    "name": "NewPolicyCreateMutation",
    "operationKind": "mutation",
    "text": "mutation NewPolicyCreateMutation(\n  $input: CreatePolicyInput!\n) {\n  createPolicy(input: $input) {\n    policy {\n      id\n      groupPath\n      ...PolicyDetailsFragment_policy\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment PolicyDetailsFragment_policy on Policy {\n  id\n  name\n  description\n  kind\n  createdBy\n  requiredApprovals\n  opaData {\n    packageSource\n    packageVersionConstraint\n    packageDigest\n    stage\n    enforcementLevel\n    speculativeRunEnforcementLevel\n  }\n  scope {\n    type\n    action\n    pattern\n  }\n  groupPath\n  allowedUsers {\n    id\n    email\n    username\n  }\n  allowedTeams {\n    id\n    name\n  }\n  allowedServiceAccounts {\n    id\n    name\n    resourcePath\n  }\n}\n"
  }
};
})();

(node as any).hash = "9094eb054c76d3973956f185f1f12091";

export default node;
