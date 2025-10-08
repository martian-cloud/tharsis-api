/**
 * @generated SignedSource<<469ea381158f48afcf84045ad48480b0>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type AppHeaderFragment$data = {
  readonly " $fragmentSpreads": FragmentRefs<"AccountMenuFragment">;
  readonly " $fragmentType": "AppHeaderFragment";
};
export type AppHeaderFragment$key = {
  readonly " $data"?: AppHeaderFragment$data;
  readonly " $fragmentSpreads": FragmentRefs<"AppHeaderFragment">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "AppHeaderFragment",
  "selections": [
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "AccountMenuFragment"
    }
  ],
  "type": "Query",
  "abstractKey": null
};

(node as any).hash = "cdacf54662a6cf4ff3d2d1649679b8e8";

export default node;
