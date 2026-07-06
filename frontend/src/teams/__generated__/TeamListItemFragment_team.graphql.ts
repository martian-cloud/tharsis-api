/**
 * @generated SignedSource<<a36d458d3715514f12552899d474412b>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TeamListItemFragment_team$data = {
  readonly description: string;
  readonly name: string;
  readonly " $fragmentType": "TeamListItemFragment_team";
};
export type TeamListItemFragment_team$key = {
  readonly " $data"?: TeamListItemFragment_team$data;
  readonly " $fragmentSpreads": FragmentRefs<"TeamListItemFragment_team">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "TeamListItemFragment_team",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "name",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "description",
      "storageKey": null
    }
  ],
  "type": "Team",
  "abstractKey": null
};

(node as any).hash = "44db259c230a8001da6ec8780098a0ac";

export default node;
