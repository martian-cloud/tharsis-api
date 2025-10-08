/**
 * @generated SignedSource<<1dff414bc36d5e03027b04c1cdfe9a19>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type RunnerListItemFragment_runner$data = {
  readonly createdBy: string;
  readonly disabled: boolean;
  readonly groupPath: string;
  readonly id: string;
  readonly metadata: {
    readonly createdAt: any;
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
      "concreteType": "ResourceMetadata",
      "kind": "LinkedField",
      "name": "metadata",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "createdAt",
          "storageKey": null
        }
      ],
      "storageKey": null
    },
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
      "name": "disabled",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "createdBy",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "groupPath",
      "storageKey": null
    }
  ],
  "type": "Runner",
  "abstractKey": null
};

(node as any).hash = "fda0c7e4bb2e45ddcd77b05ddb8f7887";

export default node;
