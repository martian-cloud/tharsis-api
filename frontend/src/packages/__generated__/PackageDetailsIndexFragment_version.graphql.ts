/**
 * @generated SignedSource<<0102557bb97062114aa0b614fdd33929>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type PackageVersionStatus = "ERRORED" | "PENDING" | "UPLOADED" | "UPLOAD_IN_PROGRESS" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type PackageDetailsIndexFragment_version$data = {
  readonly error: string | null | undefined;
  readonly id: string;
  readonly latest: boolean;
  readonly status: PackageVersionStatus;
  readonly version: string;
  readonly " $fragmentSpreads": FragmentRefs<"PackageDetailsSidebarFragment_version">;
  readonly " $fragmentType": "PackageDetailsIndexFragment_version";
};
export type PackageDetailsIndexFragment_version$key = {
  readonly " $data"?: PackageDetailsIndexFragment_version$data;
  readonly " $fragmentSpreads": FragmentRefs<"PackageDetailsIndexFragment_version">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "PackageDetailsIndexFragment_version",
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
      "name": "version",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "status",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "error",
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
      "args": null,
      "kind": "FragmentSpread",
      "name": "PackageDetailsSidebarFragment_version"
    }
  ],
  "type": "PackageVersion",
  "abstractKey": null
};

(node as any).hash = "118396437ac54cf366ee3bfcdb5325e8";

export default node;
