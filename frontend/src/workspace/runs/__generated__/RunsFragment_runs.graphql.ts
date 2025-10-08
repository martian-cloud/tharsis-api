/**
 * @generated SignedSource<<11abc3d0c77ae00cdb462aefe9d41979>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type RunsFragment_runs$data = {
  readonly fullPath: string;
  readonly " $fragmentSpreads": FragmentRefs<"CreateRunFragment_workspace" | "RunDetailsFragment_details" | "RunsIndexFragment_runs">;
  readonly " $fragmentType": "RunsFragment_runs";
};
export type RunsFragment_runs$key = {
  readonly " $data"?: RunsFragment_runs$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunsFragment_runs">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "RunsFragment_runs",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "fullPath",
      "storageKey": null
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "RunsIndexFragment_runs"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "CreateRunFragment_workspace"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "RunDetailsFragment_details"
    }
  ],
  "type": "Workspace",
  "abstractKey": null
};

(node as any).hash = "96acfe24509b1d1b59d32416597475c1";

export default node;
