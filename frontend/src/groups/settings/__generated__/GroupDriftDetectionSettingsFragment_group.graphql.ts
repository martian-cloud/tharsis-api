/**
 * @generated SignedSource<<c2e1003fe9dec23783a4bae87a908a1f>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupDriftDetectionSettingsFragment_group$data = {
  readonly driftDetectionEnabled: {
    readonly inherited: boolean;
    readonly value: boolean;
    readonly " $fragmentSpreads": FragmentRefs<"DriftDetectionSettingsFormFragment_driftDetectionEnabled">;
  };
  readonly fullPath: string;
  readonly " $fragmentType": "GroupDriftDetectionSettingsFragment_group";
};
export type GroupDriftDetectionSettingsFragment_group$key = {
  readonly " $data"?: GroupDriftDetectionSettingsFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"GroupDriftDetectionSettingsFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "GroupDriftDetectionSettingsFragment_group",
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
  "type": "Group",
  "abstractKey": null
};

(node as any).hash = "7226d9b70b7c4bdd3c594aad4cc9d3d0";

export default node;
