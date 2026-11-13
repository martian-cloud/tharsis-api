/**
 * @generated SignedSource<<c44bae4fd969d1ff2f6762dd923c7098>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type MigrateGroupDialogFragment_group$data = {
  readonly fullPath: string;
  readonly name: string;
  readonly " $fragmentType": "MigrateGroupDialogFragment_group";
};
export type MigrateGroupDialogFragment_group$key = {
  readonly " $data"?: MigrateGroupDialogFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"MigrateGroupDialogFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "MigrateGroupDialogFragment_group",
  "selections": [
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
      "name": "fullPath",
      "storageKey": null
    }
  ],
  "type": "Group",
  "abstractKey": null
};

(node as any).hash = "ef445c394cba1f90f3387a1bb72b35f1";

export default node;
