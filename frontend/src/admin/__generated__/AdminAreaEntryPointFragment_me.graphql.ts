/**
 * @generated SignedSource<<44fdb4bb6ff3f53f38a1fb00c05e566e>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type AdminAreaEntryPointFragment_me$data = {
  readonly me: {
    readonly admin?: boolean;
    readonly adminModeEnabled?: boolean;
  } | null | undefined;
  readonly " $fragmentType": "AdminAreaEntryPointFragment_me";
};
export type AdminAreaEntryPointFragment_me$key = {
  readonly " $data"?: AdminAreaEntryPointFragment_me$data;
  readonly " $fragmentSpreads": FragmentRefs<"AdminAreaEntryPointFragment_me">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "AdminAreaEntryPointFragment_me",
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
            },
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

(node as any).hash = "6288918c6778ea5e93bbddcb5b6f7411";

export default node;
