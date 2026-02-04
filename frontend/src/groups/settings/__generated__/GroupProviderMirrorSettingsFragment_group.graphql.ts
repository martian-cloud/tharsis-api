/**
 * @generated SignedSource<<77321dec621ee9f8b8cf98a064aa8af8>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupProviderMirrorSettingsFragment_group$data = {
  readonly fullPath: string;
  readonly providerMirrorEnabled: {
    readonly inherited: boolean;
    readonly value: boolean;
    readonly " $fragmentSpreads": FragmentRefs<"ProviderMirrorSettingsFormFragment_providerMirrorEnabled">;
  };
  readonly " $fragmentType": "GroupProviderMirrorSettingsFragment_group";
};
export type GroupProviderMirrorSettingsFragment_group$key = {
  readonly " $data"?: GroupProviderMirrorSettingsFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"GroupProviderMirrorSettingsFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "GroupProviderMirrorSettingsFragment_group",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "fullPath",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "NamespaceProviderMirrorEnabled",
      "kind": "LinkedField",
      "name": "providerMirrorEnabled",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "inherited",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "value",
          "storageKey": null
        },
        {
          "args": null,
          "kind": "FragmentSpread",
          "name": "ProviderMirrorSettingsFormFragment_providerMirrorEnabled"
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Group",
  "abstractKey": null
};

(node as any).hash = "e6610f54f6a3ba93d7f155524fe084d1";

export default node;
