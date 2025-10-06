/**
 * @generated SignedSource<<fbd3b54ef1c3e18bc6cc8fb8e58569e3>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type AccountMenuFragment$data = {
  readonly config: {
    readonly tharsisSupportUrl: string;
  };
  readonly me: {
    readonly admin?: boolean;
    readonly email?: string;
    readonly username?: string;
  } | null | undefined;
  readonly version: {
    readonly buildTimestamp: any;
    readonly dbMigrationDirty: boolean;
    readonly dbMigrationVersion: string;
    readonly version: string;
  };
  readonly " $fragmentType": "AccountMenuFragment";
};
export type AccountMenuFragment$key = {
  readonly " $data"?: AccountMenuFragment$data;
  readonly " $fragmentSpreads": FragmentRefs<"AccountMenuFragment">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "AccountMenuFragment",
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
              "name": "email",
              "storageKey": null
            },
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
              "name": "admin",
              "storageKey": null
            }
          ],
          "type": "User",
          "abstractKey": null
        }
      ],
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "Version",
      "kind": "LinkedField",
      "name": "version",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "version",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "dbMigrationVersion",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "dbMigrationDirty",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "buildTimestamp",
          "storageKey": null
        }
      ],
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "Config",
      "kind": "LinkedField",
      "name": "config",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "tharsisSupportUrl",
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Query",
  "abstractKey": null
};

(node as any).hash = "fad28a6d97c0d1795deec740d59d281b";

export default node;
