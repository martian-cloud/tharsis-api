/**
 * @generated SignedSource<<5df6e56c323e94ff47ad39b4254cd06d>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupRunnersListFragment_group$data = {
  readonly fullPath: string;
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

(node as any).hash = "985bda5c925d37f803c16a5dfb0091fe";

export default node;
