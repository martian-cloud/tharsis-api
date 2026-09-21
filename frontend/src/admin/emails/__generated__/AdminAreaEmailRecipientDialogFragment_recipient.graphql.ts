/**
 * @generated SignedSource<<4208e31059a82760f18307647c06efed>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type EmailDeliveryStatus = "ABANDONED" | "ACCEPTED" | "COMPLETED" | "DELAYED" | "FAILED" | "HARD_BOUNCED" | "PENDING" | "SOFT_BOUNCED" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type AdminAreaEmailRecipientDialogFragment_recipient$data = {
  readonly address: string;
  readonly attemptCount: number;
  readonly availableAt: any;
  readonly clickedAt: any | null | undefined;
  readonly complainedAt: any | null | undefined;
  readonly deliveryStatus: EmailDeliveryStatus;
  readonly failureReason: string | null | undefined;
  readonly id: string;
  readonly lastAttemptAt: any | null | undefined;
  readonly openedAt: any | null | undefined;
  readonly " $fragmentType": "AdminAreaEmailRecipientDialogFragment_recipient";
};
export type AdminAreaEmailRecipientDialogFragment_recipient$key = {
  readonly " $data"?: AdminAreaEmailRecipientDialogFragment_recipient$data;
  readonly " $fragmentSpreads": FragmentRefs<"AdminAreaEmailRecipientDialogFragment_recipient">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "AdminAreaEmailRecipientDialogFragment_recipient",
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
      "name": "deliveryStatus",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "attemptCount",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "availableAt",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "lastAttemptAt",
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
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "failureReason",
      "storageKey": null
    }
  ],
  "type": "EmailRecipient",
  "abstractKey": null
};

(node as any).hash = "3fbc93b8937a2e17cec1d7a628f9609d";

export default node;
