/**
 * @generated SignedSource<<c4e151a4a0d514f8c97f22619f0e4bb4>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type AccountMenuFragment$data = {
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
    }
  ],
  "type": "Query",
  "abstractKey": null
};

(node as any).hash = "e4846a5efb9d41d1e9a02bcb936dee83";

export default node;
