/**
 * @generated SignedSource<<7283c651e60cf892a7e42777b8af5202>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ModuleSourceLinkFragment_run$data = {
  readonly moduleSource: string | null | undefined;
  readonly moduleVersion: string | null | undefined;
  readonly " $fragmentType": "ModuleSourceLinkFragment_run";
};
export type ModuleSourceLinkFragment_run$key = {
  readonly " $data"?: ModuleSourceLinkFragment_run$data;
  readonly " $fragmentSpreads": FragmentRefs<"ModuleSourceLinkFragment_run">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ModuleSourceLinkFragment_run",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "moduleSource",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "moduleVersion",
      "storageKey": null
    }
  ],
  "type": "Run",
  "abstractKey": null
};

(node as any).hash = "fe3b7e9d50f4dcb3042d20114b4736f8";

export default node;
