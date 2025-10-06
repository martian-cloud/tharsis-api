/**
 * @generated SignedSource<<61ead87cb1374c424c4c58d765506383>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ManagedIdentityListItemFragment_managedIdentity$data = {
  readonly description: string;
  readonly groupPath: string;
  readonly id: string;
  readonly isAlias: boolean;
  readonly metadata: {
    readonly updatedAt: any;
  };
  readonly name: string;
  readonly resourcePath: string;
  readonly type: string;
  readonly " $fragmentType": "ManagedIdentityListItemFragment_managedIdentity";
};
export type ManagedIdentityListItemFragment_managedIdentity$key = {
  readonly " $data"?: ManagedIdentityListItemFragment_managedIdentity$data;
  readonly " $fragmentSpreads": FragmentRefs<"ManagedIdentityListItemFragment_managedIdentity">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ManagedIdentityListItemFragment_managedIdentity",
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
      "name": "isAlias",
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
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "groupPath",
      "storageKey": null
    }
  ],
  "type": "ManagedIdentity",
  "abstractKey": null
};

(node as any).hash = "145af6bff9ff9abda83bc35cabc9b496";

export default node;
