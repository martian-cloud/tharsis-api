/**
 * @generated SignedSource<<45215b2cbd44f3f2d7a342980d142ec6>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type EditServiceAccountFragment_group$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "EditServiceAccountFragment_group";
};
export type EditServiceAccountFragment_group$key = {
  readonly " $data"?: EditServiceAccountFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"EditServiceAccountFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "EditServiceAccountFragment_group",
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
  "type": "Group",
  "abstractKey": null
};

(node as any).hash = "12eeae7eed06663166d66175015812aa";

export default node;
