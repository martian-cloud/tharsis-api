/**
 * @generated SignedSource<<2582b4b935793a52ef0ce3f09c13d599>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type NamespaceMembershipListItemFragment_membership$data = {
  readonly id: string;
  readonly member: {
    readonly __typename: "ServiceAccount";
    readonly id: string;
    readonly name: string;
    readonly resourcePath: string;
  } | {
    readonly __typename: "Team";
    readonly id: string;
    readonly name: string;
  } | {
    readonly __typename: "User";
    readonly email: string;
    readonly id: string;
    readonly username: string;
  } | {
    // This will never be '%other', but we need some
    // value in case none of the concrete values match.
    readonly __typename: "%other";
  } | null | undefined;
  readonly metadata: {
    readonly createdAt: any;
    readonly trn: string;
    readonly updatedAt: any;
  };
  readonly resourcePath: string;
  readonly role: {
    readonly name: string;
  };
  readonly " $fragmentType": "NamespaceMembershipListItemFragment_membership";
};
export type NamespaceMembershipListItemFragment_membership$key = {
  readonly " $data"?: NamespaceMembershipListItemFragment_membership$data;
  readonly " $fragmentSpreads": FragmentRefs<"NamespaceMembershipListItemFragment_membership">;
};

const node: ReaderFragment = (function(){
var v0 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v1 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "name",
  "storageKey": null
},
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "resourcePath",
  "storageKey": null
};
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "NamespaceMembershipListItemFragment_membership",
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
          "name": "updatedAt",
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
    (v0/*:: as any*/),
    {
      "alias": null,
      "args": null,
      "concreteType": "Role",
      "kind": "LinkedField",
      "name": "role",
      "plural": false,
      "selections": [
        (v1/*:: as any*/)
      ],
      "storageKey": null
    },
    (v2/*:: as any*/),
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
            (v0/*:: as any*/),
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
            (v0/*:: as any*/),
            (v1/*:: as any*/)
          ],
          "type": "Team",
          "abstractKey": null
        },
        {
          "kind": "InlineFragment",
          "selections": [
            (v0/*:: as any*/),
            (v1/*:: as any*/),
            (v2/*:: as any*/)
          ],
          "type": "ServiceAccount",
          "abstractKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "NamespaceMembership",
  "abstractKey": null
};
})();

(node as any).hash = "cb80f9347c88c41660dfba0d0013b4e6";

export default node;
