/**
 * @generated SignedSource<<1731c0a6032f770b8f2ee003a82b450b>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type PackageKind = "OPA_POLICY" | "%future added value";
export type PackageVisibility = "GLOBAL" | "PRIVATE" | "ROOT_GROUP" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type PackageSearchListItemFragment_package$data = {
  readonly groupPath: string;
  readonly id: string;
  readonly kind: PackageKind;
  readonly latestVersion: {
    readonly createdByActor: {
      readonly " $fragmentSpreads": FragmentRefs<"ActorDisplayFragment_actor">;
    } | null | undefined;
    readonly metadata: {
      readonly createdAt: any;
    };
    readonly version: string;
  } | null | undefined;
  readonly name: string;
  readonly visibility: PackageVisibility;
  readonly " $fragmentType": "PackageSearchListItemFragment_package";
};
export type PackageSearchListItemFragment_package$key = {
  readonly " $data"?: PackageSearchListItemFragment_package$data;
  readonly " $fragmentSpreads": FragmentRefs<"PackageSearchListItemFragment_package">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "PackageSearchListItemFragment_package",
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
      "name": "name",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "groupPath",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "kind",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "visibility",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "PackageVersion",
      "kind": "LinkedField",
      "name": "latestVersion",
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
          "concreteType": null,
          "kind": "LinkedField",
          "name": "createdByActor",
          "plural": false,
          "selections": [
            {
              "args": null,
              "kind": "FragmentSpread",
              "name": "ActorDisplayFragment_actor"
            }
          ],
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
      "storageKey": null
    }
  ],
  "type": "Package",
  "abstractKey": null
};

(node as any).hash = "fce0dea61b85a9f994376a211ff45d64";

export default node;
