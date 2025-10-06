/**
 * @generated SignedSource<<aad530c9e358425a3ae702a47e9fab52>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GPGKeyListItemFragment_key$data = {
  readonly createdBy: string;
  readonly fingerprint: string;
  readonly gpgKeyId: string;
  readonly groupPath: string;
  readonly id: string;
  readonly metadata: {
    readonly createdAt: any;
    readonly trn: string;
  };
  readonly " $fragmentType": "GPGKeyListItemFragment_key";
};
export type GPGKeyListItemFragment_key$key = {
  readonly " $data"?: GPGKeyListItemFragment_key$data;
  readonly " $fragmentSpreads": FragmentRefs<"GPGKeyListItemFragment_key">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "GPGKeyListItemFragment_key",
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
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "trn",
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
      "name": "gpgKeyId",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "fingerprint",
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
      "name": "groupPath",
      "storageKey": null
    }
  ],
  "type": "GPGKey",
  "abstractKey": null
};

(node as any).hash = "dad95116312da6813b4a03b6bbb6885b";

export default node;
