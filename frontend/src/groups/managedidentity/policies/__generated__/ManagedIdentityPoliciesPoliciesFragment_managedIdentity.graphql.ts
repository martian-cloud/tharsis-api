/**
 * @generated SignedSource<<83e3b78c420bc02f6c6323170a70f2a4>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ManagedIdentityPoliciesPoliciesFragment_managedIdentity$data = {
  readonly referencingPolicies: ReadonlyArray<{
    readonly id: string;
    readonly " $fragmentSpreads": FragmentRefs<"PolicyCardFragment_policy">;
  }>;
  readonly " $fragmentType": "ManagedIdentityPoliciesPoliciesFragment_managedIdentity";
};
export type ManagedIdentityPoliciesPoliciesFragment_managedIdentity$key = {
  readonly " $data"?: ManagedIdentityPoliciesPoliciesFragment_managedIdentity$data;
  readonly " $fragmentSpreads": FragmentRefs<"ManagedIdentityPoliciesPoliciesFragment_managedIdentity">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ManagedIdentityPoliciesPoliciesFragment_managedIdentity",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "Policy",
      "kind": "LinkedField",
      "name": "referencingPolicies",
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
  "type": "ManagedIdentity",
  "abstractKey": null
};

(node as any).hash = "3b9fceb7e529632ac43c065a58316d79";

export default node;
