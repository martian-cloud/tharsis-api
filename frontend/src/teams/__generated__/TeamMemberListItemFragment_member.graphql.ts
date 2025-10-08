/**
 * @generated SignedSource<<27f5956bbfafb6ea6f525b7983bf686d>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TeamMemberListItemFragment_member$data = {
  readonly isMaintainer: boolean;
  readonly metadata: {
    readonly updatedAt: any;
  };
  readonly user: {
    readonly email: string;
    readonly username: string;
  };
  readonly " $fragmentType": "TeamMemberListItemFragment_member";
};
export type TeamMemberListItemFragment_member$key = {
  readonly " $data"?: TeamMemberListItemFragment_member$data;
  readonly " $fragmentSpreads": FragmentRefs<"TeamMemberListItemFragment_member">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "TeamMemberListItemFragment_member",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "User",
      "kind": "LinkedField",
      "name": "user",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "username",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "email",
          "storageKey": null
        }
      ],
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
          "name": "updatedAt",
          "storageKey": null
        }
      ],
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "isMaintainer",
      "storageKey": null
    }
  ],
  "type": "TeamMember",
  "abstractKey": null
};

(node as any).hash = "ebef5e162a2fc9975df1c6408e2472d1";

export default node;
