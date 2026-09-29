/**
 * @generated SignedSource<<368ccc71727b4c1143dc35ba3325429a>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type AssignedManagedIdentityListFragment_workspace$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "AssignedManagedIdentityListFragment_workspace";
};
export type AssignedManagedIdentityListFragment_workspace$key = {
  readonly " $data"?: AssignedManagedIdentityListFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"AssignedManagedIdentityListFragment_workspace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "AssignedManagedIdentityListFragment_workspace",
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
  "type": "Workspace",
  "abstractKey": null
};

(node as any).hash = "39f760366be35a664000017ad35e6c04";

export default node;
