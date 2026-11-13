/**
 * @generated SignedSource<<1a4d87e505481e99dd769c1b189514c8>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ManagedIdentityAliasesFragment_managedIdentity$data = {
  readonly " $fragmentSpreads": FragmentRefs<"ManagedIdentityAliasesListFragment_managedIdentity" | "NewManagedIdentityAliasDialogFragment_managedIdentity">;
  readonly " $fragmentType": "ManagedIdentityAliasesFragment_managedIdentity";
};
export type ManagedIdentityAliasesFragment_managedIdentity$key = {
  readonly " $data"?: ManagedIdentityAliasesFragment_managedIdentity$data;
  readonly " $fragmentSpreads": FragmentRefs<"ManagedIdentityAliasesFragment_managedIdentity">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ManagedIdentityAliasesFragment_managedIdentity",
  "selections": [
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "ManagedIdentityAliasesListFragment_managedIdentity"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "NewManagedIdentityAliasDialogFragment_managedIdentity"
    }
  ],
  "type": "ManagedIdentity",
  "abstractKey": null
};

(node as any).hash = "f55c114ffa6f7184652d4e89f8ff599d";

export default node;
