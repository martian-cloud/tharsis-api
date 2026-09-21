/**
 * @generated SignedSource<<533e0898a8069ce286f15ddbaf2e2958>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type EmailDeliveryStatus = "ABANDONED" | "ACCEPTED" | "COMPLETED" | "DELAYED" | "FAILED" | "HARD_BOUNCED" | "PENDING" | "SOFT_BOUNCED" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type AdminAreaEmailRecipientListItemFragment_recipient$data = {
  readonly address: string;
  readonly clickedAt: any | null | undefined;
  readonly complainedAt: any | null | undefined;
  readonly deliveryStatus: EmailDeliveryStatus;
  readonly openedAt: any | null | undefined;
  readonly " $fragmentSpreads": FragmentRefs<"AdminAreaEmailRecipientDialogFragment_recipient">;
  readonly " $fragmentType": "AdminAreaEmailRecipientListItemFragment_recipient";
};
export type AdminAreaEmailRecipientListItemFragment_recipient$key = {
  readonly " $data"?: AdminAreaEmailRecipientListItemFragment_recipient$data;
  readonly " $fragmentSpreads": FragmentRefs<"AdminAreaEmailRecipientListItemFragment_recipient">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "AdminAreaEmailRecipientListItemFragment_recipient",
  "selections": [
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
      "name": "deliveryStatus",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "openedAt",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "clickedAt",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "complainedAt",
      "storageKey": null
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "AdminAreaEmailRecipientDialogFragment_recipient"
    }
  ],
  "type": "EmailRecipient",
  "abstractKey": null
};

(node as any).hash = "141af145410be199821ebde6ec1149e9";

export default node;
