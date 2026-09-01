/**
 * @generated SignedSource<<2af2ba65336099b17800ef54896bb37c>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type CleanupRuleKind = "RUNS" | "TERRAFORM_MODULES" | "TERRAFORM_PROVIDERS" | "%future added value";
export type CleanupStrategy = "AGE" | "COUNT" | "PROTECT" | "%future added value";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type RunStatus = "applied" | "apply_queued" | "apply_queuing" | "applying" | "canceled" | "discarded" | "errored" | "pending" | "plan_queued" | "plan_queuing" | "planned" | "planned_and_finished" | "planning" | "post_apply_running" | "post_plan_awaiting_decision" | "post_plan_running" | "pre_apply_awaiting_decision" | "pre_apply_completed" | "pre_apply_queuing" | "pre_apply_running" | "pre_plan_awaiting_decision" | "pre_plan_completed" | "pre_plan_queuing" | "pre_plan_running" | "%future added value";
export type CreateCleanupPolicyInput = {
  clientMutationId?: string | null | undefined;
  disabled?: boolean | null | undefined;
  kind: CleanupRuleKind;
  namespacePath: string;
  runPolicyData?: RunCleanupPolicyDataInput | null | undefined;
  terraformModulePolicyData?: TerraformModuleCleanupPolicyDataInput | null | undefined;
  terraformProviderPolicyData?: TerraformProviderCleanupPolicyDataInput | null | undefined;
};
export type RunCleanupPolicyDataInput = {
  rules: ReadonlyArray<RunCleanupRuleInput>;
};
export type RunCleanupRuleInput = {
  assessment?: boolean | null | undefined;
  deleteAfterDays: number;
  description: string;
  keepMin: number;
  speculative?: boolean | null | undefined;
  status: ReadonlyArray<RunStatus>;
  strategy: CleanupStrategy;
};
export type TerraformModuleCleanupPolicyDataInput = {
  rules: ReadonlyArray<TerraformModuleCleanupRuleInput>;
};
export type TerraformModuleCleanupRuleInput = {
  deleteAfterDays: number;
  description: string;
  nameGlob: string;
  strategy: CleanupStrategy;
  systemGlob: string;
  versionGlob: string;
};
export type TerraformProviderCleanupPolicyDataInput = {
  rules: ReadonlyArray<TerraformProviderCleanupRuleInput>;
};
export type TerraformProviderCleanupRuleInput = {
  deleteAfterDays: number;
  description: string;
  nameGlob: string;
  strategy: CleanupStrategy;
  versionGlob: string;
};
export type NewCleanupPolicyCreateMutation$variables = {
  input: CreateCleanupPolicyInput;
};
export type NewCleanupPolicyCreateMutation$data = {
  readonly createCleanupPolicy: {
    readonly namespace: {
      readonly effectiveCleanupPolicies: ReadonlyArray<{
        readonly " $fragmentSpreads": FragmentRefs<"CleanupPolicyListItem_fields">;
      }>;
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type NewCleanupPolicyCreateMutation = {
  response: NewCleanupPolicyCreateMutation$data;
  variables: NewCleanupPolicyCreateMutation$variables;
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
  "name": "strategy",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "description",
  "storageKey": null
},
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "nameGlob",
  "storageKey": null
},
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "versionGlob",
  "storageKey": null
},
v8 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "deleteAfterDays",
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "NewCleanupPolicyCreateMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "CleanupPolicyMutationPayload",
        "kind": "LinkedField",
        "name": "createCleanupPolicy",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": null,
            "kind": "LinkedField",
            "name": "namespace",
            "plural": false,
            "selections": [
              {
                "alias": null,
                "args": null,
                "concreteType": "CleanupPolicy",
                "kind": "LinkedField",
                "name": "effectiveCleanupPolicies",
                "plural": true,
                "selections": [
                  {
                    "args": null,
                    "kind": "FragmentSpread",
                    "name": "CleanupPolicyListItem_fields"
                  }
                ],
                "storageKey": null
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
    "name": "NewCleanupPolicyCreateMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "CleanupPolicyMutationPayload",
        "kind": "LinkedField",
        "name": "createCleanupPolicy",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": null,
            "kind": "LinkedField",
            "name": "namespace",
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
                "alias": null,
                "args": null,
                "concreteType": "CleanupPolicy",
                "kind": "LinkedField",
                "name": "effectiveCleanupPolicies",
                "plural": true,
                "selections": [
                  (v3/*: any*/),
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
                    "name": "namespacePath",
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
                    "concreteType": "ResourceMetadata",
                    "kind": "LinkedField",
                    "name": "metadata",
                    "plural": false,
                    "selections": [
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
                    "kind": "ScalarField",
                    "name": "lastSweepCompletedAt",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "TerraformModuleCleanupPolicyData",
                    "kind": "LinkedField",
                    "name": "terraformModulePolicyData",
                    "plural": false,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "TerraformModuleCleanupRule",
                        "kind": "LinkedField",
                        "name": "rules",
                        "plural": true,
                        "selections": [
                          (v4/*: any*/),
                          (v5/*: any*/),
                          (v6/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "systemGlob",
                            "storageKey": null
                          },
                          (v7/*: any*/),
                          (v8/*: any*/)
                        ],
                        "storageKey": null
                      }
                    ],
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "TerraformProviderCleanupPolicyData",
                    "kind": "LinkedField",
                    "name": "terraformProviderPolicyData",
                    "plural": false,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "TerraformProviderCleanupRule",
                        "kind": "LinkedField",
                        "name": "rules",
                        "plural": true,
                        "selections": [
                          (v4/*: any*/),
                          (v5/*: any*/),
                          (v6/*: any*/),
                          (v7/*: any*/),
                          (v8/*: any*/)
                        ],
                        "storageKey": null
                      }
                    ],
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "RunCleanupPolicyData",
                    "kind": "LinkedField",
                    "name": "runPolicyData",
                    "plural": false,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "RunCleanupRule",
                        "kind": "LinkedField",
                        "name": "rules",
                        "plural": true,
                        "selections": [
                          (v4/*: any*/),
                          (v5/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "speculative",
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
                            "name": "status",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "keepMin",
                            "storageKey": null
                          },
                          (v8/*: any*/)
                        ],
                        "storageKey": null
                      }
                    ],
                    "storageKey": null
                  }
                ],
                "storageKey": null
              },
              (v3/*: any*/)
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
    "cacheID": "71b6ef727f2e7e9fd0e1a8f43ba83570",
    "id": null,
    "metadata": {},
    "name": "NewCleanupPolicyCreateMutation",
    "operationKind": "mutation",
    "text": "mutation NewCleanupPolicyCreateMutation(\n  $input: CreateCleanupPolicyInput!\n) {\n  createCleanupPolicy(input: $input) {\n    namespace {\n      __typename\n      effectiveCleanupPolicies {\n        ...CleanupPolicyListItem_fields\n        id\n      }\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment CleanupPolicyListItem_fields on CleanupPolicy {\n  id\n  kind\n  namespacePath\n  disabled\n  metadata {\n    trn\n  }\n  lastSweepCompletedAt\n  terraformModulePolicyData {\n    rules {\n      strategy\n      description\n      nameGlob\n      systemGlob\n      versionGlob\n      deleteAfterDays\n    }\n  }\n  terraformProviderPolicyData {\n    rules {\n      strategy\n      description\n      nameGlob\n      versionGlob\n      deleteAfterDays\n    }\n  }\n  runPolicyData {\n    rules {\n      strategy\n      description\n      speculative\n      assessment\n      status\n      keepMin\n      deleteAfterDays\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "cae75e1a6115de03b34da7d4aef6480f";

export default node;
