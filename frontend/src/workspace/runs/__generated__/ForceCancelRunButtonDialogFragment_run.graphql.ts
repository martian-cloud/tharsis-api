/**
 * @generated SignedSource<<ba569e5835586461a384511f1f36d9db>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ForceCancelRunButtonDialogFragment_run$data = {
  readonly workspace: {
    readonly fullPath: string;
  };
  readonly " $fragmentType": "ForceCancelRunButtonDialogFragment_run";
};
export type ForceCancelRunButtonDialogFragment_run$key = {
  readonly " $data"?: ForceCancelRunButtonDialogFragment_run$data;
  readonly " $fragmentSpreads": FragmentRefs<"ForceCancelRunButtonDialogFragment_run">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ForceCancelRunButtonDialogFragment_run",
  "selections": [
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

(node as any).hash = "75cfbb0239b7ce31affc3278309322dc";

export default node;
