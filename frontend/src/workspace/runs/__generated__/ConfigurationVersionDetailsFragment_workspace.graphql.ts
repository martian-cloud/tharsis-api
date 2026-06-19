/**
 * @generated SignedSource<<d930bbefbd7a9ea37c64f40595312b41>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ConfigurationVersionDetailsFragment_workspace$data = {
  readonly fullPath: string;
  readonly " $fragmentType": "ConfigurationVersionDetailsFragment_workspace";
};
export type ConfigurationVersionDetailsFragment_workspace$key = {
  readonly " $data"?: ConfigurationVersionDetailsFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"ConfigurationVersionDetailsFragment_workspace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ConfigurationVersionDetailsFragment_workspace",
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

(node as any).hash = "ea3c0deca137adc0e5787644934fdf65";

export default node;
