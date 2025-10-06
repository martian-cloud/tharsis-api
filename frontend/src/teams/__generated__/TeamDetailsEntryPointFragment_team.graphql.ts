/**
 * @generated SignedSource<<3a940293d21b0101b54cc9b4e976adfc>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { Fragment, ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TeamDetailsEntryPointFragment_team$data = {
  readonly description: string;
  readonly name: string;
  readonly " $fragmentSpreads": FragmentRefs<"TeamMemberListFragment_members">;
  readonly " $fragmentType": "TeamDetailsEntryPointFragment_team";
};
export type TeamDetailsEntryPointFragment_team$key = {
  readonly " $data"?: TeamDetailsEntryPointFragment_team$data;
  readonly " $fragmentSpreads": FragmentRefs<"TeamDetailsEntryPointFragment_team">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "TeamDetailsEntryPointFragment_team",
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
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "TeamMemberListFragment_members"
    }
  ],
  "type": "Team",
  "abstractKey": null
};

(node as any).hash = "ce6f8942d11482d10f893b04f9930892";

export default node;
