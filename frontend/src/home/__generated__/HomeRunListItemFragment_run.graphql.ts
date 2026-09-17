/**
 * @generated SignedSource<<158fde94ea61c6a3ebe0324aa3d2fb52>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type HomeRunListItemFragment_run$data = {
  readonly createdBy: string;
  readonly id: string;
  readonly isDestroy: boolean;
  readonly metadata: {
    readonly createdAt: any;
  };
  readonly workspace: {
    readonly fullPath: string;
  };
  readonly " $fragmentSpreads": FragmentRefs<"RunAnnotationsFragment_run" | "RunStageIconsFragment_run">;
  readonly " $fragmentType": "HomeRunListItemFragment_run";
};
export type HomeRunListItemFragment_run$key = {
  readonly " $data"?: HomeRunListItemFragment_run$data;
  readonly " $fragmentSpreads": FragmentRefs<"HomeRunListItemFragment_run">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "HomeRunListItemFragment_run",
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
      "name": "createdBy",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "isDestroy",
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
          "name": "createdAt",
          "storageKey": null
        }
      ],
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
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "RunStageIconsFragment_run"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "RunAnnotationsFragment_run"
    }
  ],
  "type": "Run",
  "abstractKey": null
};

(node as any).hash = "60ea0d8d461b20914e404872c5314044";

export default node;
