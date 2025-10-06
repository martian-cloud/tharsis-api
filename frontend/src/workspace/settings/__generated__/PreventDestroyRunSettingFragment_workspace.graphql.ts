/**
 * @generated SignedSource<<9a7d413b2b6e027dc80891598db61666>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { Fragment, ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type PreventDestroyRunSettingFragment_workspace$data = {
  readonly fullPath: string;
  readonly preventDestroyPlan: boolean;
  readonly " $fragmentType": "PreventDestroyRunSettingFragment_workspace";
};
export type PreventDestroyRunSettingFragment_workspace$key = {
  readonly " $data"?: PreventDestroyRunSettingFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"PreventDestroyRunSettingFragment_workspace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "PreventDestroyRunSettingFragment_workspace",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "fullPath",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "preventDestroyPlan",
      "storageKey": null
    }
  ],
  "type": "Workspace",
  "abstractKey": null
};

(node as any).hash = "d2acd97e9d6e8a393746a4a1e8416cfb";

export default node;
