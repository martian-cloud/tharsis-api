/**
 * @generated SignedSource<<94d7aa3eeab571d96abccc58b7c02571>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { Fragment, ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type RunnerListFragment_group$data = {
  readonly fullPath: string;
  readonly " $fragmentType": "RunnerListFragment_group";
};
export type RunnerListFragment_group$key = {
  readonly " $data"?: RunnerListFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunnerListFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "RunnerListFragment_group",
  "selections": [
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

(node as any).hash = "b81858367c7a6ad0d576d47cdd709436";

export default node;
