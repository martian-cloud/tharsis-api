/**
 * @generated SignedSource<<1d8d02f21842dd9f568876c3f2db5938>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type CleanupRuleKind = "RUNS" | "TERRAFORM_MODULES" | "TERRAFORM_PROVIDERS" | "%future added value";
export type CleanupStrategy = "AGE" | "COUNT" | "PROTECT" | "%future added value";
export type RunStatus = "applied" | "apply_queued" | "apply_queuing" | "applying" | "canceled" | "discarded" | "errored" | "pending" | "plan_queued" | "plan_queuing" | "planned" | "planned_and_finished" | "planning" | "post_apply_running" | "post_plan_awaiting_decision" | "post_plan_running" | "pre_apply_awaiting_decision" | "pre_apply_completed" | "pre_apply_queuing" | "pre_apply_running" | "pre_plan_awaiting_decision" | "pre_plan_completed" | "pre_plan_queuing" | "pre_plan_running" | "%future added value";
export type EditCleanupPolicyQuery$variables = {
  namespacePath: string;
};
export type EditCleanupPolicyQuery$data = {
  readonly namespace: {
    readonly __typename: string;
    readonly effectiveCleanupPolicies: ReadonlyArray<{
      readonly disabled: boolean;
      readonly id: string;
      readonly kind: CleanupRuleKind;
      readonly lastSweepCompletedAt: any | null | undefined;
      readonly namespacePath: string;
      readonly runPolicyData: {
        readonly rules: ReadonlyArray<{
          readonly assessment: boolean | null | undefined;
          readonly deleteAfterDays: number;
          readonly description: string;
          readonly keepMin: number;
          readonly speculative: boolean | null | undefined;
          readonly status: ReadonlyArray<RunStatus>;
          readonly strategy: CleanupStrategy;
        }>;
      } | null | undefined;
      readonly terraformModulePolicyData: {
        readonly rules: ReadonlyArray<{
          readonly deleteAfterDays: number;
          readonly description: string;
          readonly nameGlob: string;
          readonly strategy: CleanupStrategy;
          readonly systemGlob: string;
          readonly versionGlob: string;
        }>;
      } | null | undefined;
      readonly terraformProviderPolicyData: {
        readonly rules: ReadonlyArray<{
          readonly deleteAfterDays: number;
          readonly description: string;
          readonly nameGlob: string;
          readonly strategy: CleanupStrategy;
          readonly versionGlob: string;
        }>;
      } | null | undefined;
    }>;
    readonly fullPath: string;
    readonly id: string;
  } | null | undefined;
};
export type EditCleanupPolicyQuery = {
  response: EditCleanupPolicyQuery$data;
  variables: EditCleanupPolicyQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "namespacePath"
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
  "name": "strategy",
  "storageKey": null
},
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "description",
  "storageKey": null
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "nameGlob",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "versionGlob",
  "storageKey": null
},
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "deleteAfterDays",
  "storageKey": null
},
v7 = [
  {
    "alias": null,
    "args": [
      {
        "kind": "Variable",
        "name": "fullPath",
        "variableName": "namespacePath"
      }
    ],
    "concreteType": null,
    "kind": "LinkedField",
    "name": "namespace",
    "plural": false,
    "selections": [
      (v1/*: any*/),
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
        "kind": "ScalarField",
        "name": "fullPath",
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
          (v1/*: any*/),
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
                  (v2/*: any*/),
                  (v3/*: any*/),
                  (v4/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "systemGlob",
                    "storageKey": null
                  },
                  (v5/*: any*/),
                  (v6/*: any*/)
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
                  (v2/*: any*/),
                  (v3/*: any*/),
                  (v4/*: any*/),
                  (v5/*: any*/),
                  (v6/*: any*/)
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
                  (v2/*: any*/),
                  (v3/*: any*/),
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
                  (v6/*: any*/)
                ],
                "storageKey": null
              }
            ],
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
    "name": "EditCleanupPolicyQuery",
    "selections": (v7/*: any*/),
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "EditCleanupPolicyQuery",
    "selections": (v7/*: any*/)
  },
  "params": {
    "cacheID": "7d61d6730fa1f459a34d687733662983",
    "id": null,
    "metadata": {},
    "name": "EditCleanupPolicyQuery",
    "operationKind": "query",
    "text": "query EditCleanupPolicyQuery(\n  $namespacePath: String!\n) {\n  namespace(fullPath: $namespacePath) {\n    id\n    __typename\n    fullPath\n    effectiveCleanupPolicies {\n      id\n      kind\n      namespacePath\n      disabled\n      lastSweepCompletedAt\n      terraformModulePolicyData {\n        rules {\n          strategy\n          description\n          nameGlob\n          systemGlob\n          versionGlob\n          deleteAfterDays\n        }\n      }\n      terraformProviderPolicyData {\n        rules {\n          strategy\n          description\n          nameGlob\n          versionGlob\n          deleteAfterDays\n        }\n      }\n      runPolicyData {\n        rules {\n          strategy\n          description\n          speculative\n          assessment\n          status\n          keepMin\n          deleteAfterDays\n        }\n      }\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "5e1bb91c6852680d3a4b9f734b4c1cfc";

export default node;
