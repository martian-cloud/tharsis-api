/**
 * @generated SignedSource<<3727ab0feb65bd4899f83736eec1d19a>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ProviderMirrorListItemFragment_mirror$data = {
  readonly createdByActor: {
    readonly " $fragmentSpreads": FragmentRefs<"ActorDisplayFragment_actor">;
  } | null | undefined;
  readonly id: string;
  readonly metadata: {
    readonly createdAt: any;
  };
  readonly providerAddress: string;
  readonly version: string;
  readonly " $fragmentType": "ProviderMirrorListItemFragment_mirror";
};
export type ProviderMirrorListItemFragment_mirror$key = {
  readonly " $data"?: ProviderMirrorListItemFragment_mirror$data;
  readonly " $fragmentSpreads": FragmentRefs<"ProviderMirrorListItemFragment_mirror">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ProviderMirrorListItemFragment_mirror",
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
      "name": "providerAddress",
      "storageKey": null
    }
  ],
  "type": "TerraformProviderVersionMirror",
  "abstractKey": null
};

(node as any).hash = "f11398dfff71e94a62aec68b0687f7a2";

export default node;
