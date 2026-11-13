/**
 * @generated SignedSource<<7c6388f2d5dc66ac925de4d2bbdba38c>>
 * @lightSyntaxTransform
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
