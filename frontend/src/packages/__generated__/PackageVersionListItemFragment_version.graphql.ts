/**
 * @generated SignedSource<<8e99c5524f711c6829ffa060d01266cc>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type PackageVersionStatus = "ERRORED" | "PENDING" | "UPLOADED" | "UPLOAD_IN_PROGRESS" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type PackageVersionListItemFragment_version$data = {
  readonly createdBy: string;
  readonly id: string;
  readonly latest: boolean;
  readonly metadata: {
    readonly createdAt: any;
  };
  readonly status: PackageVersionStatus;
  readonly version: string;
  readonly " $fragmentType": "PackageVersionListItemFragment_version";
};
export type PackageVersionListItemFragment_version$key = {
  readonly " $data"?: PackageVersionListItemFragment_version$data;
  readonly " $fragmentSpreads": FragmentRefs<"PackageVersionListItemFragment_version">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "PackageVersionListItemFragment_version",
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
          "name": "createdAt",
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
      "name": "version",
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
      "name": "latest",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "status",
      "storageKey": null
    }
  ],
  "type": "PackageVersion",
  "abstractKey": null
};

(node as any).hash = "7e3564bf166bbd5ea3ce7f613a30c630";

export default node;
