/**
 * @generated SignedSource<<f69aa92929bd4431991df254c7834e01>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type CleanupRuleKind = "RUNS" | "TERRAFORM_MODULES" | "TERRAFORM_PROVIDERS" | "%future added value";
export type CleanupStrategy = "AGE" | "COUNT" | "PROTECT" | "%future added value";
export type RunStatus = "applied" | "apply_queued" | "apply_queuing" | "applying" | "canceled" | "discarded" | "errored" | "pending" | "plan_queued" | "plan_queuing" | "planned" | "planned_and_finished" | "planning" | "post_apply_running" | "post_plan_awaiting_decision" | "post_plan_running" | "pre_apply_awaiting_decision" | "pre_apply_completed" | "pre_apply_queuing" | "pre_apply_running" | "pre_plan_awaiting_decision" | "pre_plan_completed" | "pre_plan_queuing" | "pre_plan_running" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type CleanupPolicyListItem_fields$data = {
  readonly disabled: boolean;
  readonly id: string;
  readonly kind: CleanupRuleKind;
  readonly lastSweepCompletedAt: any | null | undefined;
  readonly metadata: {
    readonly trn: string;
  };
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
  readonly " $fragmentType": "CleanupPolicyListItem_fields";
};
export type CleanupPolicyListItem_fields$key = {
  readonly " $data"?: CleanupPolicyListItem_fields$data;
  readonly " $fragmentSpreads": FragmentRefs<"CleanupPolicyListItem_fields">;
};

const node: ReaderFragment = (function(){
var v0 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "strategy",
  "storageKey": null
},
v1 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "description",
  "storageKey": null
},
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "nameGlob",
  "storageKey": null
},
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "versionGlob",
  "storageKey": null
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "deleteAfterDays",
  "storageKey": null
};
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "CleanupPolicyListItem_fields",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "id",
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
            (v0/*: any*/),
            (v1/*: any*/),
            (v2/*: any*/),
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "systemGlob",
              "storageKey": null
            },
            (v3/*: any*/),
            (v4/*: any*/)
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
            (v0/*: any*/),
            (v1/*: any*/),
            (v2/*: any*/),
            (v3/*: any*/),
            (v4/*: any*/)
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
            (v0/*: any*/),
            (v1/*: any*/),
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
            (v4/*: any*/)
          ],
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "CleanupPolicy",
  "abstractKey": null
};
})();

(node as any).hash = "a74646f4d1562dc8ed723b740cf03dda";

export default node;
