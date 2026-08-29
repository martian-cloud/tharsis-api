/**
 * @generated SignedSource<<bcd17f4645367d042a20b246e55fda29>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type PolicyEnforcementLevel = "ADVISORY" | "HARD_MANDATORY" | "SOFT_MANDATORY" | "%future added value";
export type PolicyKind = "OPA" | "%future added value";
export type PolicyScopeRuleAction = "EXCLUDE" | "INCLUDE" | "%future added value";
export type PolicyStage = "POST_APPLY" | "POST_PLAN" | "PRE_APPLY" | "PRE_PLAN" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type PolicyCardFragment_policy$data = {
  readonly allowedServiceAccounts: ReadonlyArray<{
    readonly id: string;
  }>;
  readonly allowedTeams: ReadonlyArray<{
    readonly id: string;
  }>;
  readonly allowedUsers: ReadonlyArray<{
    readonly id: string;
  }>;
  readonly createdBy: string;
  readonly description: string;
  readonly groupPath: string;
  readonly id: string;
  readonly kind: PolicyKind;
  readonly name: string;
  readonly opaData: {
    readonly enforcementLevel: PolicyEnforcementLevel;
    readonly packageSource: string;
    readonly packageVersionConstraint: string | null | undefined;
    readonly stage: PolicyStage;
  } | null | undefined;
  readonly requiredApprovals: number;
  readonly scope: ReadonlyArray<{
    readonly action: PolicyScopeRuleAction;
  }>;
  readonly " $fragmentType": "PolicyCardFragment_policy";
};
export type PolicyCardFragment_policy$key = {
  readonly " $data"?: PolicyCardFragment_policy$data;
  readonly " $fragmentSpreads": FragmentRefs<"PolicyCardFragment_policy">;
};

const node: ReaderFragment = (function(){
var v0 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v1 = [
  (v0/*: any*/)
];
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "PolicyCardFragment_policy",
  "selections": [
    (v0/*: any*/),
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "name",
      "storageKey": null
    },
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
          "name": "stage",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "enforcementLevel",
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
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "action",
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
      "selections": (v1/*: any*/),
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "Team",
      "kind": "LinkedField",
      "name": "allowedTeams",
      "plural": true,
      "selections": (v1/*: any*/),
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "ServiceAccount",
      "kind": "LinkedField",
      "name": "allowedServiceAccounts",
      "plural": true,
      "selections": (v1/*: any*/),
      "storageKey": null
    }
  ],
  "type": "Policy",
  "abstractKey": null
};
})();

(node as any).hash = "0c68963a8d592a4053fa3247fdf1219f";

export default node;
