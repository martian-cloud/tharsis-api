/**
 * @generated SignedSource<<94fb68587a646fcc405136850cd5fe71>>
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
export type PackageDetailsSidebarFragment_package$data = {
  readonly allowMutableVersions: boolean;
  readonly groupPath: string;
  readonly kind: PackageKind;
  readonly visibility: PackageVisibility;
  readonly " $fragmentType": "PackageDetailsSidebarFragment_package";
};
export type PackageDetailsSidebarFragment_package$key = {
  readonly " $data"?: PackageDetailsSidebarFragment_package$data;
  readonly " $fragmentSpreads": FragmentRefs<"PackageDetailsSidebarFragment_package">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "PackageDetailsSidebarFragment_package",
  "selections": [
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
      "name": "kind",
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
      "name": "allowMutableVersions",
      "storageKey": null
    }
  ],
  "type": "Package",
  "abstractKey": null
};

(node as any).hash = "b87afdbea7b374a12ba85f2a535d8f32";

export default node;
