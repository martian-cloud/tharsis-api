/**
 * @generated SignedSource<<c6e5af11ddad4af55fba905b9392797f>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupRunnersFragment_group$data = {
  readonly " $fragmentSpreads": FragmentRefs<"EditGroupRunnerFragment_group" | "GroupRunnerDetailsFragment_group" | "GroupRunnersListFragment_group" | "NewGroupRunnerFragment_group">;
  readonly " $fragmentType": "GroupRunnersFragment_group";
};
export type GroupRunnersFragment_group$key = {
  readonly " $data"?: GroupRunnersFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"GroupRunnersFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "GroupRunnersFragment_group",
  "selections": [
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "GroupRunnersListFragment_group"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "NewGroupRunnerFragment_group"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "EditGroupRunnerFragment_group"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "GroupRunnerDetailsFragment_group"
    }
  ],
  "type": "Group",
  "abstractKey": null
};

(node as any).hash = "05c202b0bec8c898c564e62d3a0897cf";

export default node;
