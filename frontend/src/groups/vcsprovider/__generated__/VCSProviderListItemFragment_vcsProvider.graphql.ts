/**
 * @generated SignedSource<<1e2502d3ceef1472f6a3a6cd90ddb379>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type VCSProviderListItemFragment_vcsProvider$data = {
  readonly description: string;
  readonly groupPath: string;
  readonly id: string;
  readonly metadata: {
    readonly updatedAt: any;
  };
  readonly name: string;
  readonly " $fragmentType": "VCSProviderListItemFragment_vcsProvider";
};
export type VCSProviderListItemFragment_vcsProvider$key = {
  readonly " $data"?: VCSProviderListItemFragment_vcsProvider$data;
  readonly " $fragmentSpreads": FragmentRefs<"VCSProviderListItemFragment_vcsProvider">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "VCSProviderListItemFragment_vcsProvider",
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
      "name": "groupPath",
      "storageKey": null
    }
  ],
  "type": "VCSProvider",
  "abstractKey": null
};

(node as any).hash = "50490040cb0e749849e7c0e07983f470";

export default node;
