/**
 * @generated SignedSource<<3b43300d68c7739b65638a9b143d76f5>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupRunnerSettingsFragment_group$data = {
  readonly fullPath: string;
  readonly runnerTags: {
    readonly inherited: boolean;
    readonly namespacePath: string;
    readonly value: ReadonlyArray<string>;
    readonly " $fragmentSpreads": FragmentRefs<"RunnerSettingsForm_runnerTags">;
  };
  readonly " $fragmentType": "GroupRunnerSettingsFragment_group";
};
export type GroupRunnerSettingsFragment_group$key = {
  readonly " $data"?: GroupRunnerSettingsFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"GroupRunnerSettingsFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "GroupRunnerSettingsFragment_group",
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
      "concreteType": "NamespaceRunnerTags",
      "kind": "LinkedField",
      "name": "runnerTags",
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
          "name": "namespacePath",
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
          "name": "RunnerSettingsForm_runnerTags"
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Group",
  "abstractKey": null
};

(node as any).hash = "5134275246b00d5b32eddc8f4b2151c0";

export default node;
