/**
 * @generated SignedSource<<9c70303ff80a65106c7290f184c8decc>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ActivityEventListItemFragment_event$data = {
  readonly id: string;
  readonly initiator: {
    readonly __typename: "ServiceAccount";
    readonly name: string;
    readonly resourcePath: string;
  } | {
    readonly __typename: "User";
    readonly email: string;
    readonly username: string;
  } | {
    // This will never be '%other', but we need some
    // value in case none of the concrete values match.
    readonly __typename: "%other";
  };
  readonly metadata: {
    readonly createdAt: any;
  };
  readonly " $fragmentType": "ActivityEventListItemFragment_event";
};
export type ActivityEventListItemFragment_event$key = {
  readonly " $data"?: ActivityEventListItemFragment_event$data;
  readonly " $fragmentSpreads": FragmentRefs<"ActivityEventListItemFragment_event">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ActivityEventListItemFragment_event",
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
      "concreteType": null,
      "kind": "LinkedField",
      "name": "initiator",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "__typename",
          "storageKey": null
        },
        {
          "kind": "InlineFragment",
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
          "type": "User",
          "abstractKey": null
        },
        {
          "kind": "InlineFragment",
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
              "name": "resourcePath",
              "storageKey": null
            }
          ],
          "type": "ServiceAccount",
          "abstractKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "ActivityEvent",
  "abstractKey": null
};

(node as any).hash = "b4cbb452f8092160c4fb6401c8691617";

export default node;
