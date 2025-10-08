/**
 * @generated SignedSource<<19f11f7f4f082fe046cbb5fa772440cb>>
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

(node as any).hash = "12d0c4ac8b2e300df4508b9b64aebdec";

export default node;
