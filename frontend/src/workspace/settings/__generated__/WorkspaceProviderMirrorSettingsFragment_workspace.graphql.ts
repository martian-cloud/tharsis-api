/**
 * @generated SignedSource<<ad851d2baeaf2baa3255273ec5bf843c>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type WorkspaceProviderMirrorSettingsFragment_workspace$data = {
  readonly fullPath: string;
  readonly providerMirrorEnabled: {
    readonly inherited: boolean;
    readonly value: boolean;
    readonly " $fragmentSpreads": FragmentRefs<"ProviderMirrorSettingsFormFragment_providerMirrorEnabled">;
  };
  readonly " $fragmentType": "WorkspaceProviderMirrorSettingsFragment_workspace";
};
export type WorkspaceProviderMirrorSettingsFragment_workspace$key = {
  readonly " $data"?: WorkspaceProviderMirrorSettingsFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceProviderMirrorSettingsFragment_workspace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "WorkspaceProviderMirrorSettingsFragment_workspace",
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
  "type": "Workspace",
  "abstractKey": null
};

(node as any).hash = "daea724dcdb026385cb61cc76e96fc47";

export default node;
