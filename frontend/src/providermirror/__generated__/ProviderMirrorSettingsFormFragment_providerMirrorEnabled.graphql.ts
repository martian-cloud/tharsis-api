/**
 * @generated SignedSource<<625ff3b2839e457e0c6d49f094bc27b4>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ProviderMirrorSettingsFormFragment_providerMirrorEnabled$data = {
  readonly inherited: boolean;
  readonly namespacePath: string;
  readonly value: boolean;
  readonly " $fragmentType": "ProviderMirrorSettingsFormFragment_providerMirrorEnabled";
};
export type ProviderMirrorSettingsFormFragment_providerMirrorEnabled$key = {
  readonly " $data"?: ProviderMirrorSettingsFormFragment_providerMirrorEnabled$data;
  readonly " $fragmentSpreads": FragmentRefs<"ProviderMirrorSettingsFormFragment_providerMirrorEnabled">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ProviderMirrorSettingsFormFragment_providerMirrorEnabled",
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
      "name": "namespacePath",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "value",
      "storageKey": null
    }
  ],
  "type": "NamespaceProviderMirrorEnabled",
  "abstractKey": null
};

(node as any).hash = "43f175bfbd3c10efab1964e24619be4f";

export default node;
