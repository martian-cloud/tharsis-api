/**
 * @generated SignedSource<<eb64d275bd77d74eed65e46290936597>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type NewGroupRunnerFragment_group$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "NewGroupRunnerFragment_group";
};
export type NewGroupRunnerFragment_group$key = {
  readonly " $data"?: NewGroupRunnerFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"NewGroupRunnerFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "NewGroupRunnerFragment_group",
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

(node as any).hash = "4ef850e2c56b13a65baca8fa2185d7e6";

export default node;
