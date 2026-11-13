/**
 * @generated SignedSource<<dcabd4b26c153a48c19e905c41a941bf>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type NewVCSProviderLinkFragment_workspace$data = {
  readonly fullPath: string;
  readonly " $fragmentSpreads": FragmentRefs<"VCSProviderLinkFormFragment_workspace">;
  readonly " $fragmentType": "NewVCSProviderLinkFragment_workspace";
};
export type NewVCSProviderLinkFragment_workspace$key = {
  readonly " $data"?: NewVCSProviderLinkFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"NewVCSProviderLinkFragment_workspace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "NewVCSProviderLinkFragment_workspace",
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
      "name": "VCSProviderLinkFormFragment_workspace"
    }
  ],
  "type": "Workspace",
  "abstractKey": null
};

(node as any).hash = "a11c1ecf095a142a90b8cd31fa80b29e";

export default node;
