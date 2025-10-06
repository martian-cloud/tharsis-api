/**
 * @generated SignedSource<<712b7560ff35bce3909572da062b8b7b>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TeamDetailsFragment_team$data = {
  readonly description: string;
  readonly metadata: {
    readonly trn: string;
  };
  readonly name: string;
  readonly " $fragmentSpreads": FragmentRefs<"TeamMemberListFragment_members">;
  readonly " $fragmentType": "TeamDetailsFragment_team";
};
export type TeamDetailsFragment_team$key = {
  readonly " $data"?: TeamDetailsFragment_team$data;
  readonly " $fragmentSpreads": FragmentRefs<"TeamDetailsFragment_team">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "TeamDetailsFragment_team",
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
      "alias": null,
      "args": null,
      "concreteType": "ResourceMetadata",
      "kind": "LinkedField",
      "name": "metadata",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "trn",
          "storageKey": null
        }
      ],
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

(node as any).hash = "8e69e7805f7a1ac5c1fc3f1377ab95d7";

export default node;
