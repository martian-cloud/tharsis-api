/**
 * @generated SignedSource<<d400e24dacf6d76688791d241c386212>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type HomeTeamListItemFragment_team$data = {
  readonly name: string;
  readonly " $fragmentType": "HomeTeamListItemFragment_team";
};
export type HomeTeamListItemFragment_team$key = {
  readonly " $data"?: HomeTeamListItemFragment_team$data;
  readonly " $fragmentSpreads": FragmentRefs<"HomeTeamListItemFragment_team">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "HomeTeamListItemFragment_team",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "name",
      "storageKey": null
    }
  ],
  "type": "Team",
  "abstractKey": null
};

(node as any).hash = "c509fa9b19e7fec92fc56136d0f370d6";

export default node;
