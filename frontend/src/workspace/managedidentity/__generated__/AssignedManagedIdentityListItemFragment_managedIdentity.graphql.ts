/**
 * @generated SignedSource<<e3da63e8a5edcf3e17e9279deb3d7f89>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type AssignedManagedIdentityListItemFragment_managedIdentity$data = {
  readonly description: string;
  readonly id: string;
  readonly metadata: {
    readonly updatedAt: any;
  };
  readonly name: string;
  readonly resourcePath: string;
  readonly type: string;
  readonly " $fragmentType": "AssignedManagedIdentityListItemFragment_managedIdentity";
};
export type AssignedManagedIdentityListItemFragment_managedIdentity$key = {
  readonly " $data"?: AssignedManagedIdentityListItemFragment_managedIdentity$data;
  readonly " $fragmentSpreads": FragmentRefs<"AssignedManagedIdentityListItemFragment_managedIdentity">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "AssignedManagedIdentityListItemFragment_managedIdentity",
  "selections": [
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
          "name": "updatedAt",
          "storageKey": null
        }
      ],
      "storageKey": null
    },
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
      "name": "name",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "description",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "type",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "resourcePath",
      "storageKey": null
    }
  ],
  "type": "ManagedIdentity",
  "abstractKey": null
};

(node as any).hash = "337b8a4b8b4e1d76298265c045ee02be";

export default node;
