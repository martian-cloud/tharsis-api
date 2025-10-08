/**
 * @generated SignedSource<<0ac492e99cfe776446bd9a9104cc7a26>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type NewManagedIdentityFragment_group$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "NewManagedIdentityFragment_group";
};
export type NewManagedIdentityFragment_group$key = {
  readonly " $data"?: NewManagedIdentityFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"NewManagedIdentityFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "NewManagedIdentityFragment_group",
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
      "name": "fullPath",
      "storageKey": null
    }
  ],
  "type": "Group",
  "abstractKey": null
};

(node as any).hash = "63cb40f5616e408aa76c96a03881c446";

export default node;
