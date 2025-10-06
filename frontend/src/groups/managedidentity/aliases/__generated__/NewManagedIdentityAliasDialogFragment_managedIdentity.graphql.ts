/**
 * @generated SignedSource<<a7e0f9c945fab1a03865105b7ee71fe5>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type NewManagedIdentityAliasDialogFragment_managedIdentity$data = {
  readonly groupPath: string;
  readonly id: string;
  readonly " $fragmentType": "NewManagedIdentityAliasDialogFragment_managedIdentity";
};
export type NewManagedIdentityAliasDialogFragment_managedIdentity$key = {
  readonly " $data"?: NewManagedIdentityAliasDialogFragment_managedIdentity$data;
  readonly " $fragmentSpreads": FragmentRefs<"NewManagedIdentityAliasDialogFragment_managedIdentity">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "NewManagedIdentityAliasDialogFragment_managedIdentity",
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
      "name": "groupPath",
      "storageKey": null
    }
  ],
  "type": "ManagedIdentity",
  "abstractKey": null
};

(node as any).hash = "bc14350b5bffed5c7e9d4632b9668625";

export default node;
