/**
 * @generated SignedSource<<46f3c90e3e86f988a69299b901ad97d7>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupRunnersListFragment_group$data = {
  readonly id: string;
  readonly " $fragmentType": "GroupRunnersListFragment_group";
};
export type GroupRunnersListFragment_group$key = {
  readonly " $data"?: GroupRunnersListFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"GroupRunnersListFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "GroupRunnersListFragment_group",
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

(node as any).hash = "2383b0737d318af12127c6070e07998c";

export default node;
