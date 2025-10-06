/**
 * @generated SignedSource<<12c61ce76e31082fbaf0840d07e46664>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type WorkspaceDriftDetectionSettingsFragment_workspace$data = {
  readonly driftDetectionEnabled: {
    readonly inherited: boolean;
    readonly value: boolean;
    readonly " $fragmentSpreads": FragmentRefs<"DriftDetectionSettingsFormFragment_driftDetectionEnabled">;
  };
  readonly fullPath: string;
  readonly " $fragmentType": "WorkspaceDriftDetectionSettingsFragment_workspace";
};
export type WorkspaceDriftDetectionSettingsFragment_workspace$key = {
  readonly " $data"?: WorkspaceDriftDetectionSettingsFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceDriftDetectionSettingsFragment_workspace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "WorkspaceDriftDetectionSettingsFragment_workspace",
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
      "concreteType": "NamespaceDriftDetectionEnabled",
      "kind": "LinkedField",
      "name": "driftDetectionEnabled",
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
          "name": "DriftDetectionSettingsFormFragment_driftDetectionEnabled"
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Workspace",
  "abstractKey": null
};

(node as any).hash = "006d06c2690999d55869c1e0631e5080";

export default node;
