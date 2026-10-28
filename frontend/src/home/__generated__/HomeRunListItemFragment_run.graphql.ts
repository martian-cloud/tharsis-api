/**
 * @generated SignedSource<<0917b8102fc01667da3c1178dead7b52>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type HomeRunListItemFragment_run$data = {
  readonly createdByActor: {
    readonly " $fragmentSpreads": FragmentRefs<"ActorDisplayFragment_actor">;
  } | null | undefined;
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
      "concreteType": null,
      "kind": "LinkedField",
      "name": "createdByActor",
      "plural": false,
      "selections": [
        {
          "args": null,
          "kind": "FragmentSpread",
          "name": "ActorDisplayFragment_actor"
        }
      ],
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

(node as any).hash = "b6160184b97156f3258ec4acddeb2b06";

export default node;
