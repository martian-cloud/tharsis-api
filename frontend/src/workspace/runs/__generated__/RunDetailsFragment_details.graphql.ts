/**
 * @generated SignedSource<<711b546eff2091ee9156347c542b5ba4>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type RunDetailsFragment_details$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "RunDetailsFragment_details";
};
export type RunDetailsFragment_details$key = {
  readonly " $data"?: RunDetailsFragment_details$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunDetailsFragment_details">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "RunDetailsFragment_details",
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
  "type": "Workspace",
  "abstractKey": null
};

(node as any).hash = "850710076b3df241ba0a3bfce755f963";

export default node;
