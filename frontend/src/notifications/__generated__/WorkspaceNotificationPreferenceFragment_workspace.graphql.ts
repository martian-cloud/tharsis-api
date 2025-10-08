/**
 * @generated SignedSource<<4c24882082cc0d1e486f3aaaceb2346c>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type WorkspaceNotificationPreferenceFragment_workspace$data = {
  readonly fullPath: string;
  readonly " $fragmentType": "WorkspaceNotificationPreferenceFragment_workspace";
};
export type WorkspaceNotificationPreferenceFragment_workspace$key = {
  readonly " $data"?: WorkspaceNotificationPreferenceFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceNotificationPreferenceFragment_workspace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "WorkspaceNotificationPreferenceFragment_workspace",
  "selections": [
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

(node as any).hash = "b360d8059a186eb3b40ab70aabeeedcc";

export default node;
