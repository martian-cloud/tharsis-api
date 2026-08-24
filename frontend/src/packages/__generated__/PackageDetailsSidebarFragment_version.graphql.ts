/**
 * @generated SignedSource<<6c4c23ce749f8b22d536a73fe7c15f57>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type PackageDetailsSidebarFragment_version$data = {
  readonly createdBy: string;
  readonly latest: boolean;
  readonly metadata: {
    readonly createdAt: any;
  };
  readonly shaSum: string;
  readonly size: number;
  readonly version: string;
  readonly " $fragmentType": "PackageDetailsSidebarFragment_version";
};
export type PackageDetailsSidebarFragment_version$key = {
  readonly " $data"?: PackageDetailsSidebarFragment_version$data;
  readonly " $fragmentSpreads": FragmentRefs<"PackageDetailsSidebarFragment_version">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "PackageDetailsSidebarFragment_version",
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
      "name": "latest",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "createdBy",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "shaSum",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "size",
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
  "type": "PackageVersion",
  "abstractKey": null
};

(node as any).hash = "49e311f88a7d87b898b8b28b923827ac";

export default node;
