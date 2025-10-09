/**
 * @generated SignedSource<<6cb1f1f511701bffc107fd7a05e2c2c5>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type UserSessionFragment_session$data = {
  readonly current: boolean;
  readonly expiration: any;
  readonly expired: boolean;
  readonly id: string;
  readonly metadata: {
    readonly createdAt: any;
  };
  readonly userAgent: string;
  readonly " $fragmentType": "UserSessionFragment_session";
};
export type UserSessionFragment_session$key = {
  readonly " $data"?: UserSessionFragment_session$data;
  readonly " $fragmentSpreads": FragmentRefs<"UserSessionFragment_session">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "UserSessionFragment_session",
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
      "name": "userAgent",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "expiration",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "expired",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "current",
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
  "type": "UserSession",
  "abstractKey": null
};

(node as any).hash = "6672ada216ad1720ac98313f05f8da71";

export default node;
