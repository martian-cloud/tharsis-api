/**
 * @generated SignedSource<<3c98fc5d27adcfc29ad80f0b31bd010d>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type WorkspaceListFragment_group$data = {
  readonly id: string;
  readonly " $fragmentType": "WorkspaceListFragment_group";
};
export type WorkspaceListFragment_group$key = {
  readonly " $data"?: WorkspaceListFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceListFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "WorkspaceListFragment_group",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "id",
      "storageKey": null
    }
  ],
  "type": "Group",
  "abstractKey": null
};

(node as any).hash = "d21280794ae681444de9ffd712f548ac";

export default node;
