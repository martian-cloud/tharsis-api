/**
 * @generated SignedSource<<de286a8837ef509ef5598a039fc03964>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupTreeContainerFragment_me$data = {
  readonly me: {
    readonly admin?: boolean;
  } | null | undefined;
  readonly " $fragmentType": "GroupTreeContainerFragment_me";
};
export type GroupTreeContainerFragment_me$key = {
  readonly " $data"?: GroupTreeContainerFragment_me$data;
  readonly " $fragmentSpreads": FragmentRefs<"GroupTreeContainerFragment_me">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "GroupTreeContainerFragment_me",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": null,
      "kind": "LinkedField",
      "name": "me",
      "plural": false,
      "selections": [
        {
          "kind": "InlineFragment",
          "selections": [
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "admin",
              "storageKey": null
            }
          ],
          "type": "User",
          "abstractKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Query",
  "abstractKey": null
};

(node as any).hash = "9b2ec4ef3e8a368e1c21071091757c12";

export default node;
