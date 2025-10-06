/**
 * @generated SignedSource<<5b4833e460df7dbf4e21989e551e53b4>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type RunnerSettingsForm_runnerTags$data = {
  readonly inherited: boolean;
  readonly namespacePath: string;
  readonly value: ReadonlyArray<string>;
  readonly " $fragmentType": "RunnerSettingsForm_runnerTags";
};
export type RunnerSettingsForm_runnerTags$key = {
  readonly " $data"?: RunnerSettingsForm_runnerTags$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunnerSettingsForm_runnerTags">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "RunnerSettingsForm_runnerTags",
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
  "type": "NamespaceRunnerTags",
  "abstractKey": null
};

(node as any).hash = "dbbfca7166b6b9d8d80877c53183f140";

export default node;
