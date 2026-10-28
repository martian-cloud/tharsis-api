/**
 * @generated SignedSource<<77557f5d07cb32e7ea66ef24b8913665>>
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
  readonly createdByActor: {
    readonly " $fragmentSpreads": FragmentRefs<"ActorDisplayFragment_actor">;
  } | null | undefined;
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

(node as any).hash = "b212d2b8a1dc5f4f8469ceb0537d77cd";

export default node;
