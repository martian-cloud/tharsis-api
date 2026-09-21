/**
 * @generated SignedSource<<b06a2ca289591b64b963cb8ceb46ed15>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type AdminAreaEmailOutboxListItemFragment_outbox$data = {
  readonly emailType: string;
  readonly ephemeral: boolean;
  readonly id: string;
  readonly metadata: {
    readonly createdAt: any;
  };
  readonly subject: string;
  readonly " $fragmentType": "AdminAreaEmailOutboxListItemFragment_outbox";
};
export type AdminAreaEmailOutboxListItemFragment_outbox$key = {
  readonly " $data"?: AdminAreaEmailOutboxListItemFragment_outbox$data;
  readonly " $fragmentSpreads": FragmentRefs<"AdminAreaEmailOutboxListItemFragment_outbox">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "AdminAreaEmailOutboxListItemFragment_outbox",
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
      "name": "subject",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "emailType",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "ephemeral",
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
        }
      ],
      "storageKey": null
    }
  ],
  "type": "EmailOutboxItem",
  "abstractKey": null
};

(node as any).hash = "c13d4b2fb85f8c9bec3daf5250a276c2";

export default node;
