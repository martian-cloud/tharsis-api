/**
 * @generated SignedSource<<9b46a61941f4e8bcc59c44c3b6906593>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type AssignedServiceAccountListFragment_runner$data = {
  readonly id: string;
  readonly resourcePath: string;
  readonly " $fragmentType": "AssignedServiceAccountListFragment_runner";
};
export type AssignedServiceAccountListFragment_runner$key = {
  readonly " $data"?: AssignedServiceAccountListFragment_runner$data;
  readonly " $fragmentSpreads": FragmentRefs<"AssignedServiceAccountListFragment_runner">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "AssignedServiceAccountListFragment_runner",
  "selections": [
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
      "name": "resourcePath",
      "storageKey": null
    }
  ],
  "type": "Runner",
  "abstractKey": null
};

(node as any).hash = "afdbd55921e6bed95f88aee3e7aaaf2f";

export default node;
