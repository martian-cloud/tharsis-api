/**
 * @generated SignedSource<<90ba362a6cb60a31dad09bc1e20395cf>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type AssignedPolicyListPoliciesFragment_workspace$data = {
  readonly assignedPolicies: ReadonlyArray<{
    readonly id: string;
    readonly " $fragmentSpreads": FragmentRefs<"PolicyCardFragment_policy">;
  }>;
  readonly " $fragmentType": "AssignedPolicyListPoliciesFragment_workspace";
};
export type AssignedPolicyListPoliciesFragment_workspace$key = {
  readonly " $data"?: AssignedPolicyListPoliciesFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"AssignedPolicyListPoliciesFragment_workspace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "AssignedPolicyListPoliciesFragment_workspace",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "Policy",
      "kind": "LinkedField",
      "name": "assignedPolicies",
      "plural": true,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "id",
          "storageKey": null
        },
        {
          "args": null,
          "kind": "FragmentSpread",
          "name": "PolicyCardFragment_policy"
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Workspace",
  "abstractKey": null
};

(node as any).hash = "c1c94f290b07915defc4015a8112293a";

export default node;
