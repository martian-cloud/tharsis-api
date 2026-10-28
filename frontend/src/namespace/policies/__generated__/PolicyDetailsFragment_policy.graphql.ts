/**
 * @generated SignedSource<<628b56406105fbb932e185f78523b525>>
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
export type PolicyScopeRuleType = "GROUP" | "MANAGED_IDENTITY" | "WORKSPACE" | "%future added value";
export type PolicyStage = "POST_APPLY" | "POST_PLAN" | "PRE_APPLY" | "PRE_PLAN" | "%future added value";
export type SpeculativeRunEnforcementLevel = "ADVISORY" | "HARD_MANDATORY" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type PolicyDetailsFragment_policy$data = {
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
  readonly " $fragmentType": "PolicyDetailsFragment_policy";
};
export type PolicyDetailsFragment_policy$key = {
  readonly " $data"?: PolicyDetailsFragment_policy$data;
  readonly " $fragmentSpreads": FragmentRefs<"PolicyDetailsFragment_policy">;
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
  "name": "name",
  "storageKey": null
},
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "stage",
  "storageKey": null
},
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "enforcementLevel",
  "storageKey": null
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "speculativeRunEnforcementLevel",
  "storageKey": null
};
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "PolicyDetailsFragment_policy",
  "selections": [
    (v0/*: any*/),
    (v1/*: any*/),
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
        (v2/*: any*/),
        (v3/*: any*/),
        (v4/*: any*/)
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
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "verifyStateLineage",
          "storageKey": null
        },
        (v2/*: any*/),
        (v3/*: any*/),
        (v4/*: any*/)
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
          "name": "type",
          "storageKey": null
        },
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
        (v0/*: any*/),
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
        (v0/*: any*/),
        (v1/*: any*/)
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
        (v0/*: any*/),
        (v1/*: any*/),
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
  "type": "Policy",
  "abstractKey": null
};
})();

(node as any).hash = "d84fca594027702b47704dd67ed417b0";

export default node;
