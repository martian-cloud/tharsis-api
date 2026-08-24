/**
 * @generated SignedSource<<10cbf07dac9c331fb0bc06bd0a02ed85>>
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
  readonly name: string;
  readonly packageSource: string;
  readonly packageVersionConstraint: string;
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

(node as any).hash = "de644c0e5ababd7b3d60e35a006a31ce";

export default node;
