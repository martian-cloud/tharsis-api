/**
 * @generated SignedSource<<ca9d0bc058bc06e78c90f5c08af6a3dd>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type WorkspaceRoleBindingFragment_workspace$data = {
  readonly fullPath: string;
  readonly groupPath: string;
  readonly id: string;
  readonly name: string;
  readonly " $fragmentType": "WorkspaceRoleBindingFragment_workspace";
};
export type WorkspaceRoleBindingFragment_workspace$key = {
  readonly " $data"?: WorkspaceRoleBindingFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceRoleBindingFragment_workspace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "WorkspaceRoleBindingFragment_workspace",
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
      "name": "fullPath",
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
  "type": "Workspace",
  "abstractKey": null
};

(node as any).hash = "afcd3576833ce8e053076f2912f3990a";

export default node;
