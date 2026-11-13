/**
 * @generated SignedSource<<821ae5d6e94d74f952a5ea976e811349>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupTreeContainerFragment_me$data = {
  readonly me: {
    readonly adminModeEnabled?: boolean;
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
              "name": "adminModeEnabled",
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

(node as any).hash = "6ae848ebd40c7846cc0df9109ca41a0e";

export default node;
