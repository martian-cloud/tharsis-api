/**
 * @generated SignedSource<<b4a3da6720ff25bb234d7bcc6b3c0494>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type NamespaceMembershipListFragment_memberships$data = {
  readonly fullPath: string;
  readonly memberships: ReadonlyArray<{
    readonly id: string;
    readonly member: {
      readonly __typename: "ServiceAccount";
      readonly name: string;
      readonly resourcePath: string;
    } | {
      readonly __typename: "Team";
      readonly name: string;
    } | {
      readonly __typename: "User";
      readonly email: string;
      readonly username: string;
    } | {
      // This will never be '%other', but we need some
      // value in case none of the concrete values match.
      readonly __typename: "%other";
    } | null | undefined;
    readonly " $fragmentSpreads": FragmentRefs<"NamespaceMembershipListItemFragment_membership">;
  }>;
  readonly " $fragmentType": "NamespaceMembershipListFragment_memberships";
};
export type NamespaceMembershipListFragment_memberships$key = {
  readonly " $data"?: NamespaceMembershipListFragment_memberships$data;
  readonly " $fragmentSpreads": FragmentRefs<"NamespaceMembershipListFragment_memberships">;
};

const node: ReaderFragment = (function(){
var v0 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "name",
  "storageKey": null
};
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "NamespaceMembershipListFragment_memberships",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "fullPath",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "NamespaceMembership",
      "kind": "LinkedField",
      "name": "memberships",
      "plural": true,
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
          "concreteType": null,
          "kind": "LinkedField",
          "name": "member",
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
                (v0/*: any*/)
              ],
              "type": "Team",
              "abstractKey": null
            },
            {
              "kind": "InlineFragment",
              "selections": [
                {
                  "alias": null,
                  "args": null,
                  "kind": "ScalarField",
                  "name": "resourcePath",
                  "storageKey": null
                },
                (v0/*: any*/)
              ],
              "type": "ServiceAccount",
              "abstractKey": null
            }
          ],
          "storageKey": null
        },
        {
          "args": null,
          "kind": "FragmentSpread",
          "name": "NamespaceMembershipListItemFragment_membership"
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Namespace",
  "abstractKey": "__isNamespace"
};
})();

(node as any).hash = "722c24e26a4824dbf89c17317529dfad";

export default node;
