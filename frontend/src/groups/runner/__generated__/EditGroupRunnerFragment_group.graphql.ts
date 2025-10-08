/**
 * @generated SignedSource<<36dce109d266a3d57b2b80ceee5dc501>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type EditGroupRunnerFragment_group$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "EditGroupRunnerFragment_group";
};
export type EditGroupRunnerFragment_group$key = {
  readonly " $data"?: EditGroupRunnerFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"EditGroupRunnerFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "EditGroupRunnerFragment_group",
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

(node as any).hash = "2003d3bffb69c75bf270a3407dd6bc7b";

export default node;
