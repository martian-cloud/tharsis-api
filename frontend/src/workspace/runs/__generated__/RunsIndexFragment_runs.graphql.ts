/**
 * @generated SignedSource<<6a045b16139cb3348b91fb3562de03e9>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type RunsIndexFragment_runs$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "RunsIndexFragment_runs";
};
export type RunsIndexFragment_runs$key = {
  readonly " $data"?: RunsIndexFragment_runs$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunsIndexFragment_runs">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "RunsIndexFragment_runs",
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
      "name": "fullPath",
      "storageKey": null
    }
  ],
  "type": "Workspace",
  "abstractKey": null
};

(node as any).hash = "6e6644d2e5302170b1eec6f1b01d1337";

export default node;
