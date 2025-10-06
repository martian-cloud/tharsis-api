/**
 * @generated SignedSource<<6f1a1919da7187d8af3bcfb674767baa>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type MoveManagedIdentityDialogFragment_managedIdentity$data = {
  readonly groupPath: string;
  readonly id: string;
  readonly name: string;
  readonly " $fragmentType": "MoveManagedIdentityDialogFragment_managedIdentity";
};
export type MoveManagedIdentityDialogFragment_managedIdentity$key = {
  readonly " $data"?: MoveManagedIdentityDialogFragment_managedIdentity$data;
  readonly " $fragmentSpreads": FragmentRefs<"MoveManagedIdentityDialogFragment_managedIdentity">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "MoveManagedIdentityDialogFragment_managedIdentity",
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
      "name": "name",
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

(node as any).hash = "7cd08741a9f1aa7aed89f66f96a62a07";

export default node;
