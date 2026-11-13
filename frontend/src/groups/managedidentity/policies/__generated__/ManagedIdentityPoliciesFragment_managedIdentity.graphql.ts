/**
 * @generated SignedSource<<3a64be744664ae0db1037623a64c6a92>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ManagedIdentityPoliciesFragment_managedIdentity$data = {
  readonly id: string;
  readonly " $fragmentType": "ManagedIdentityPoliciesFragment_managedIdentity";
};
export type ManagedIdentityPoliciesFragment_managedIdentity$key = {
  readonly " $data"?: ManagedIdentityPoliciesFragment_managedIdentity$data;
  readonly " $fragmentSpreads": FragmentRefs<"ManagedIdentityPoliciesFragment_managedIdentity">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ManagedIdentityPoliciesFragment_managedIdentity",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "id",
      "storageKey": null
    }
  ],
  "type": "ManagedIdentity",
  "abstractKey": null
};

(node as any).hash = "44d3dd72d4f578c53ca197e051669add";

export default node;
