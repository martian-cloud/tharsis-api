/**
 * @generated SignedSource<<f4043d71f57700dcb86fe17c0159283f>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type PolicyEnforcementLevel = "ADVISORY" | "HARD_MANDATORY" | "SOFT_MANDATORY" | "%future added value";
export type PolicyKind = "MODULE_ATTESTATION" | "OPA" | "%future added value";
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
  readonly moduleAttestationData: {
    readonly enforcementLevel: PolicyEnforcementLevel;
    readonly predicateType: string | null | undefined;
    readonly publicKey: string;
    readonly stage: PolicyStage;
  } | null | undefined;
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
v1 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "stage",
  "storageKey": null
},
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "enforcementLevel",
  "storageKey": null
},
v3 = [
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
        (v1/*: any*/),
        (v2/*: any*/)
      ],
      "storageKey": null
    },
    {
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
        (v1/*: any*/),
        (v2/*: any*/)
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
      "selections": (v3/*: any*/),
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "Team",
      "kind": "LinkedField",
      "name": "allowedTeams",
      "plural": true,
      "selections": (v3/*: any*/),
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "ServiceAccount",
      "kind": "LinkedField",
      "name": "allowedServiceAccounts",
      "plural": true,
      "selections": (v3/*: any*/),
      "storageKey": null
    }
  ],
  "type": "Policy",
  "abstractKey": null
};
})();

(node as any).hash = "ad7acc569391ee3e991cee9acd57e766";

export default node;
