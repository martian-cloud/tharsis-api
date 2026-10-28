/**
 * @generated SignedSource<<be003b5814113a7779c41035fa056fac>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TerraformProviderVersionListItemFragment_version$data = {
  readonly createdByActor: {
    readonly " $fragmentSpreads": FragmentRefs<"ActorDisplayFragment_actor">;
  } | null | undefined;
  readonly id: string;
  readonly latest: boolean;
  readonly metadata: {
    readonly createdAt: any;
  };
  readonly provider: {
    readonly name: string;
    readonly registryNamespace: string;
  };
  readonly version: string;
  readonly " $fragmentType": "TerraformProviderVersionListItemFragment_version";
};
export type TerraformProviderVersionListItemFragment_version$key = {
  readonly " $data"?: TerraformProviderVersionListItemFragment_version$data;
  readonly " $fragmentSpreads": FragmentRefs<"TerraformProviderVersionListItemFragment_version">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "TerraformProviderVersionListItemFragment_version",
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
      "concreteType": "TerraformProvider",
      "kind": "LinkedField",
      "name": "provider",
      "plural": false,
      "selections": [
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
          "name": "registryNamespace",
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "TerraformProviderVersion",
  "abstractKey": null
};

(node as any).hash = "59cd8b5c55f38e58b88fbeb689a065b0";

export default node;
