/**
 * @generated SignedSource<<b94fa5c0b47d0e5b9f3078b193f60e21>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ForceCancelRunButtonFragment_run$data = {
  readonly id: string;
  readonly workspace: {
    readonly fullPath: string;
  };
  readonly " $fragmentType": "ForceCancelRunButtonFragment_run";
};
export type ForceCancelRunButtonFragment_run$key = {
  readonly " $data"?: ForceCancelRunButtonFragment_run$data;
  readonly " $fragmentSpreads": FragmentRefs<"ForceCancelRunButtonFragment_run">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ForceCancelRunButtonFragment_run",
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
      "concreteType": "Workspace",
      "kind": "LinkedField",
      "name": "workspace",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "fullPath",
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Run",
  "abstractKey": null
};

(node as any).hash = "3849b9bd49adf07bea4d993d4e259f21";

export default node;
