/**
 * @generated SignedSource<<09af96d9f33f09d3e89cf9bef8ae21b0>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ModuleSourceFragment_workspace$data = {
  readonly fullPath: string;
  readonly " $fragmentType": "ModuleSourceFragment_workspace";
};
export type ModuleSourceFragment_workspace$key = {
  readonly " $data"?: ModuleSourceFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"ModuleSourceFragment_workspace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ModuleSourceFragment_workspace",
  "selections": [
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

(node as any).hash = "883f6cbb4125c8f75a08edc5d1855874";

export default node;
