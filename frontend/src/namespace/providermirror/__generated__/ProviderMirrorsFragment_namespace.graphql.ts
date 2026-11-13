/**
 * @generated SignedSource<<b2bdbc0550e812120e094f63637d55ce>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ProviderMirrorsFragment_namespace$data = {
  readonly fullPath: string;
  readonly " $fragmentType": "ProviderMirrorsFragment_namespace";
};
export type ProviderMirrorsFragment_namespace$key = {
  readonly " $data"?: ProviderMirrorsFragment_namespace$data;
  readonly " $fragmentSpreads": FragmentRefs<"ProviderMirrorsFragment_namespace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ProviderMirrorsFragment_namespace",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "fullPath",
      "storageKey": null
    }
  ],
  "type": "Namespace",
  "abstractKey": "__isNamespace"
};

(node as any).hash = "6fa0c29a64ad696da2a0e95ffd8e8032";

export default node;
