/**
 * @generated SignedSource<<0f8b64e5c3e493ed440fcdd7794601ca>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type EmailSuppressionCause = "COMPLAINT" | "HARD_BOUNCE" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type AdminAreaEmailSuppressionListItemFragment_suppression$data = {
  readonly address: string;
  readonly cause: EmailSuppressionCause;
  readonly id: string;
  readonly metadata: {
    readonly createdAt: any;
    readonly trn: string;
  };
  readonly " $fragmentType": "AdminAreaEmailSuppressionListItemFragment_suppression";
};
export type AdminAreaEmailSuppressionListItemFragment_suppression$key = {
  readonly " $data"?: AdminAreaEmailSuppressionListItemFragment_suppression$data;
  readonly " $fragmentSpreads": FragmentRefs<"AdminAreaEmailSuppressionListItemFragment_suppression">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "AdminAreaEmailSuppressionListItemFragment_suppression",
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
      "kind": "ScalarField",
      "name": "address",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "cause",
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
    }
  ],
  "type": "EmailSuppression",
  "abstractKey": null
};

(node as any).hash = "8c258dabe2ddb1d48a27447b4fdf8543";

export default node;
