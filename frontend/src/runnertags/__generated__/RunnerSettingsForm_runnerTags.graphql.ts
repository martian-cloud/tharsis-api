/**
 * @generated SignedSource<<646e000b020e1bc034f5a526b084c473>>
 * @lightSyntaxTransform
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
