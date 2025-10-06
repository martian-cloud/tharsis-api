/**
 * @generated SignedSource<<b6517b3f580fc709984c1711f4c6c75e>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type AdminAreaUserListItemFragment_user$data = {
  readonly active: boolean;
  readonly admin: boolean;
  readonly email: string;
  readonly id: string;
  readonly metadata: {
    readonly createdAt: any;
    readonly trn: string;
  };
  readonly scimExternalId: string | null | undefined;
  readonly username: string;
  readonly " $fragmentType": "AdminAreaUserListItemFragment_user";
};
export type AdminAreaUserListItemFragment_user$key = {
  readonly " $data"?: AdminAreaUserListItemFragment_user$data;
  readonly " $fragmentSpreads": FragmentRefs<"AdminAreaUserListItemFragment_user">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "AdminAreaUserListItemFragment_user",
  "selections": [
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
          "name": "createdAt",
          "storageKey": null
        },
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
      "name": "username",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "email",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "admin",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "active",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "scimExternalId",
      "storageKey": null
    }
  ],
  "type": "User",
  "abstractKey": null
};

(node as any).hash = "1ecebf7fb862080727266628789e0bc7";

export default node;
