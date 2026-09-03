/**
 * @generated SignedSource<<dd300b97d38959776c450b1d554aedd9>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type PolicyEnforcementLevel = "ADVISORY" | "HARD_MANDATORY" | "SOFT_MANDATORY" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type RunTaskStagePolicyCheckPolicyCardFragment_policy$data = {
  readonly description: string;
  readonly enforcementLevel: PolicyEnforcementLevel;
  readonly id: string;
  readonly messages: ReadonlyArray<string>;
  readonly moduleAttestationData: {
    readonly predicateType: string | null | undefined;
    readonly publicKey: string;
    readonly verifyStateLineage: boolean;
  } | null | undefined;
  readonly name: string;
  readonly opaData: {
    readonly packageSource: string;
    readonly packageVersionConstraint: string;
  } | null | undefined;
  readonly policy: {
    readonly groupPath: string;
    readonly id: string;
  } | null | undefined;
  readonly provenance: {
    readonly policyTrn: string;
  };
  readonly status: string;
  readonly " $fragmentType": "RunTaskStagePolicyCheckPolicyCardFragment_policy";
};
export type RunTaskStagePolicyCheckPolicyCardFragment_policy$key = {
  readonly " $data"?: RunTaskStagePolicyCheckPolicyCardFragment_policy$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunTaskStagePolicyCheckPolicyCardFragment_policy">;
};

const node: ReaderFragment = (function(){
var v0 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
};
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "RunTaskStagePolicyCheckPolicyCardFragment_policy",
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
      "concreteType": "PolicyCheckOPAData",
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
        }
      ],
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "PolicyCheckModuleAttestationData",
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
        }
      ],
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
      "name": "status",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "messages",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "PolicyCheckPolicyProvenance",
      "kind": "LinkedField",
      "name": "provenance",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "policyTrn",
          "storageKey": null
        }
      ],
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "Policy",
      "kind": "LinkedField",
      "name": "policy",
      "plural": false,
      "selections": [
        (v0/*: any*/),
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "groupPath",
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "PolicyCheckPolicy",
  "abstractKey": null
};
})();

(node as any).hash = "41e25af0eed7af33093a0d775040c2c8";

export default node;
