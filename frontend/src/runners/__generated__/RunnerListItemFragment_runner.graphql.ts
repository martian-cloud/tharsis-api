/**
 * @generated SignedSource<<0f3295185d126ad957f95e902e624a70>>
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
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "updatedAt",
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

(node as any).hash = "38f1bb36691c48e6d5a53bf27e719f07";

export default node;
