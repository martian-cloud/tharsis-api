/**
 * @generated SignedSource<<68cf733aa667aaa9584011338b106fb5>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type AssignedPolicyListFragment_workspace$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "AssignedPolicyListFragment_workspace";
};
export type AssignedPolicyListFragment_workspace$key = {
  readonly " $data"?: AssignedPolicyListFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"AssignedPolicyListFragment_workspace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "AssignedPolicyListFragment_workspace",
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

(node as any).hash = "d3e451fe7bb247a96fdb33d0f87e768d";

export default node;
