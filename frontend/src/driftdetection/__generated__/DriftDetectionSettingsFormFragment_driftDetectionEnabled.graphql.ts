/**
 * @generated SignedSource<<d6efba70e694c7e0e589d71cee0207ea>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type DriftDetectionSettingsFormFragment_driftDetectionEnabled$data = {
  readonly inherited: boolean;
  readonly namespacePath: string;
  readonly value: boolean;
  readonly " $fragmentType": "DriftDetectionSettingsFormFragment_driftDetectionEnabled";
};
export type DriftDetectionSettingsFormFragment_driftDetectionEnabled$key = {
  readonly " $data"?: DriftDetectionSettingsFormFragment_driftDetectionEnabled$data;
  readonly " $fragmentSpreads": FragmentRefs<"DriftDetectionSettingsFormFragment_driftDetectionEnabled">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "DriftDetectionSettingsFormFragment_driftDetectionEnabled",
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
  "type": "NamespaceDriftDetectionEnabled",
  "abstractKey": null
};

(node as any).hash = "88db791b7282b273c64acf4411bf769b";

export default node;
