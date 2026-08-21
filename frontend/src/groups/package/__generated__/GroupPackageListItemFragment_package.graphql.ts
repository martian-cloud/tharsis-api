/**
 * @generated SignedSource<<60879c796127048b10d4e60d3561ba82>>
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
export type GroupPackageListItemFragment_package$data = {
  readonly description: string;
  readonly groupPath: string;
  readonly id: string;
  readonly kind: PackageKind;
  readonly latestVersion: {
    readonly version: string;
  } | null | undefined;
  readonly name: string;
  readonly visibility: PackageVisibility;
  readonly " $fragmentType": "GroupPackageListItemFragment_package";
};
export type GroupPackageListItemFragment_package$key = {
  readonly " $data"?: GroupPackageListItemFragment_package$data;
  readonly " $fragmentSpreads": FragmentRefs<"GroupPackageListItemFragment_package">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "GroupPackageListItemFragment_package",
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
      "name": "description",
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
      "kind": "ScalarField",
      "name": "groupPath",
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
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Package",
  "abstractKey": null
};

(node as any).hash = "8d693466967a6c0cc19671ec2e44dc55";

export default node;
