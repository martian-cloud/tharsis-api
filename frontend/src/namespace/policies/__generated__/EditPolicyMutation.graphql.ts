/**
 * @generated SignedSource<<fcb4db9c415a5944ae6e0258b95b8a18>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type PolicyEnforcementLevel = "ADVISORY" | "HARD_MANDATORY" | "SOFT_MANDATORY" | "%future added value";
export type PolicyKind = "MODULE_ATTESTATION" | "OPA" | "%future added value";
export type PolicyScopeRuleAction = "EXCLUDE" | "INCLUDE" | "%future added value";
export type PolicyScopeRuleType = "GROUP" | "MANAGED_IDENTITY" | "WORKSPACE" | "%future added value";
export type PolicyStage = "POST_APPLY" | "POST_PLAN" | "PRE_APPLY" | "PRE_PLAN" | "%future added value";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type SpeculativeRunEnforcementLevel = "ADVISORY" | "HARD_MANDATORY" | "%future added value";
export type UpdatePolicyInput = {
  allowedServiceAccounts?: ReadonlyArray<string> | null | undefined;
  allowedTeams?: ReadonlyArray<string> | null | undefined;
  allowedUsers?: ReadonlyArray<string> | null | undefined;
  clientMutationId?: string | null | undefined;
  description?: string | null | undefined;
  id: string;
  moduleAttestationData?: ModuleAttestationPolicyDataInput | null | undefined;
  opaData?: OPAPolicyDataInput | null | undefined;
  requiredApprovals?: number | null | undefined;
  scope?: ReadonlyArray<PolicyScopeRuleInput> | null | undefined;
};
export type ModuleAttestationPolicyDataInput = {
  enforcementLevel: PolicyEnforcementLevel;
  predicateType?: string | null | undefined;
  publicKey: string;
  speculativeRunEnforcementLevel: SpeculativeRunEnforcementLevel;
  stage: PolicyStage;
  verifyStateLineage: boolean;
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
      readonly createdByActor: {
        readonly " $fragmentSpreads": FragmentRefs<"ActorDisplayFragment_actor">;
      } | null | undefined;
      readonly description: string;
      readonly groupPath: string;
      readonly id: string;
      readonly kind: PolicyKind;
      readonly moduleAttestationData: {
        readonly enforcementLevel: PolicyEnforcementLevel;
        readonly predicateType: string | null | undefined;
        readonly publicKey: string;
        readonly speculativeRunEnforcementLevel: SpeculativeRunEnforcementLevel;
        readonly stage: PolicyStage;
        readonly verifyStateLineage: boolean;
      } | null | undefined;
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
  "name": "name",
  "storageKey": null
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "description",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "kind",
  "storageKey": null
},
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "stage",
  "storageKey": null
},
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "enforcementLevel",
  "storageKey": null
},
v8 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "speculativeRunEnforcementLevel",
  "storageKey": null
},
v9 = {
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
    (v6/*: any*/),
    (v7/*: any*/),
    (v8/*: any*/)
  ],
  "storageKey": null
},
v10 = {
  "alias": null,
  "args": null,
  "concreteType": "ModuleAttestationPolicyData",
  "kind": "LinkedField",
  "name": "moduleAttestationData",
  "plural": false,
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
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "verifyStateLineage",
      "storageKey": null
    },
    (v6/*: any*/),
    (v7/*: any*/),
    (v8/*: any*/)
  ],
  "storageKey": null
},
v11 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "createdBy",
  "storageKey": null
},
v12 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "requiredApprovals",
  "storageKey": null
},
v13 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "type",
  "storageKey": null
},
v14 = {
  "alias": null,
  "args": null,
  "concreteType": "PolicyScopeRule",
  "kind": "LinkedField",
  "name": "scope",
  "plural": true,
  "selections": [
    (v13/*: any*/),
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
v15 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "groupPath",
  "storageKey": null
},
v16 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "email",
  "storageKey": null
},
v17 = {
  "alias": null,
  "args": null,
  "concreteType": "User",
  "kind": "LinkedField",
  "name": "allowedUsers",
  "plural": true,
  "selections": [
    (v2/*: any*/),
    (v16/*: any*/),
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
v18 = {
  "alias": null,
  "args": null,
  "concreteType": "Team",
  "kind": "LinkedField",
  "name": "allowedTeams",
  "plural": true,
  "selections": [
    (v2/*: any*/),
    (v3/*: any*/)
  ],
  "storageKey": null
},
v19 = {
  "alias": null,
  "args": null,
  "concreteType": "ServiceAccount",
  "kind": "LinkedField",
  "name": "allowedServiceAccounts",
  "plural": true,
  "selections": [
    (v2/*: any*/),
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
v20 = {
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
    (v13/*: any*/)
  ],
  "storageKey": null
},
v21 = [
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "trnSnapshot",
    "storageKey": null
  }
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "EditPolicyMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
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
              (v2/*: any*/),
              (v3/*: any*/),
              (v4/*: any*/),
              (v5/*: any*/),
              (v9/*: any*/),
              (v10/*: any*/),
              (v11/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": null,
                "kind": "LinkedField",
                "name": "createdByActor",
                "plural": false,
                "selections": [
                  {
                    "args": null,
                    "kind": "FragmentSpread",
                    "name": "ActorDisplayFragment_actor"
                  }
                ],
                "storageKey": null
              },
              (v12/*: any*/),
              (v14/*: any*/),
              (v15/*: any*/),
              (v17/*: any*/),
              (v18/*: any*/),
              (v19/*: any*/)
            ],
            "storageKey": null
          },
          (v20/*: any*/)
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
    "name": "EditPolicyMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
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
              (v2/*: any*/),
              (v3/*: any*/),
              (v4/*: any*/),
              (v5/*: any*/),
              (v9/*: any*/),
              (v10/*: any*/),
              (v11/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": null,
                "kind": "LinkedField",
                "name": "createdByActor",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "__typename",
                    "storageKey": null
                  },
                  {
                    "kind": "TypeDiscriminator",
                    "abstractKey": "__isActor"
                  },
                  (v13/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "principalId",
                    "storageKey": null
                  },
                  {
                    "kind": "InlineFragment",
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "User",
                        "kind": "LinkedField",
                        "name": "user",
                        "plural": false,
                        "selections": [
                          (v16/*: any*/),
                          (v2/*: any*/)
                        ],
                        "storageKey": null
                      }
                    ],
                    "type": "UserActor",
                    "abstractKey": null
                  },
                  {
                    "kind": "InlineFragment",
                    "selections": (v21/*: any*/),
                    "type": "ServiceAccountActor",
                    "abstractKey": null
                  },
                  {
                    "kind": "InlineFragment",
                    "selections": (v21/*: any*/),
                    "type": "VCSProviderActor",
                    "abstractKey": null
                  },
                  {
                    "kind": "InlineFragment",
                    "selections": (v21/*: any*/),
                    "type": "JobActor",
                    "abstractKey": null
                  }
                ],
                "storageKey": null
              },
              (v12/*: any*/),
              (v14/*: any*/),
              (v15/*: any*/),
              (v17/*: any*/),
              (v18/*: any*/),
              (v19/*: any*/)
            ],
            "storageKey": null
          },
          (v20/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "d827d9eb74717e536d9d6c9bfa9278fe",
    "id": null,
    "metadata": {},
    "name": "EditPolicyMutation",
    "operationKind": "mutation",
    "text": "mutation EditPolicyMutation(\n  $input: UpdatePolicyInput!\n) {\n  updatePolicy(input: $input) {\n    policy {\n      id\n      name\n      description\n      kind\n      opaData {\n        packageSource\n        packageVersionConstraint\n        packageDigest\n        stage\n        enforcementLevel\n        speculativeRunEnforcementLevel\n      }\n      moduleAttestationData {\n        publicKey\n        predicateType\n        verifyStateLineage\n        stage\n        enforcementLevel\n        speculativeRunEnforcementLevel\n      }\n      createdBy\n      createdByActor {\n        __typename\n        ...ActorDisplayFragment_actor\n      }\n      requiredApprovals\n      scope {\n        type\n        action\n        pattern\n      }\n      groupPath\n      allowedUsers {\n        id\n        email\n        username\n      }\n      allowedTeams {\n        id\n        name\n      }\n      allowedServiceAccounts {\n        id\n        name\n        resourcePath\n      }\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment ActorDisplayFragment_actor on Actor {\n  __isActor: __typename\n  type\n  principalId\n  ... on UserActor {\n    user {\n      email\n      id\n    }\n  }\n  ... on ServiceAccountActor {\n    trnSnapshot\n  }\n  ... on VCSProviderActor {\n    trnSnapshot\n  }\n  ... on JobActor {\n    trnSnapshot\n  }\n}\n"
  }
};
})();

(node as any).hash = "99e75bad88837563055cd11959d40ad0";

export default node;
