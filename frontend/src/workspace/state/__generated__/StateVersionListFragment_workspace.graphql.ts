/**
 * @generated SignedSource<<733e0d531c00493423ea923bfdcb238f>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type StateVersionListFragment_workspace$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "StateVersionListFragment_workspace";
};
export type StateVersionListFragment_workspace$key = {
  readonly " $data"?: StateVersionListFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"StateVersionListFragment_workspace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "StateVersionListFragment_workspace",
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

(node as any).hash = "b3c69515a35665a2eace8dc61eeca785";

export default node;
