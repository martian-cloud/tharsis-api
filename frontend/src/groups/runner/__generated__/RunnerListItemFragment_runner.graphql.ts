/**
 * @generated SignedSource<<ccf54ccef62067686e07344976f84c7f>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { Fragment, ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type RunnerListItemFragment_runner$data = {
  readonly description: string;
  readonly groupPath: string;
  readonly id: string;
  readonly metadata: {
    readonly updatedAt: any;
  };
  readonly name: string;
  readonly " $fragmentType": "RunnerListItemFragment_runner";
};
export type RunnerListItemFragment_runner$key = {
  readonly " $data"?: RunnerListItemFragment_runner$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunnerListItemFragment_runner">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "RunnerListItemFragment_runner",
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
      "name": "name",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "description",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "groupPath",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "ResourceMetadata",
      "kind": "LinkedField",
      "name": "metadata",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "updatedAt",
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Runner",
  "abstractKey": null
};

(node as any).hash = "1f60636dd6192d0338d005d19a252465";

export default node;
