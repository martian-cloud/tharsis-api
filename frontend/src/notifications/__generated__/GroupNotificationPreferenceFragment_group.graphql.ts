/**
 * @generated SignedSource<<28c875c9f244e20f922b05c242dffb4f>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupNotificationPreferenceFragment_group$data = {
  readonly fullPath: string;
  readonly " $fragmentType": "GroupNotificationPreferenceFragment_group";
};
export type GroupNotificationPreferenceFragment_group$key = {
  readonly " $data"?: GroupNotificationPreferenceFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"GroupNotificationPreferenceFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "GroupNotificationPreferenceFragment_group",
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

(node as any).hash = "c59637ed884711f985e24b6577d2232b";

export default node;
