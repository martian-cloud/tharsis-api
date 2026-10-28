/**
 * @generated SignedSource<<6835f8442fdca258a07daffff06dadcc>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type RunnerListItemFragment_runner$data = {
  readonly createdByActor: {
    readonly " $fragmentSpreads": FragmentRefs<"ActorDisplayFragment_actor">;
  } | null | undefined;
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
      "name": "groupPath",
      "storageKey": null
    }
  ],
  "type": "Runner",
  "abstractKey": null
};

(node as any).hash = "9620274dc340e4ddcef76ee651c500d0";

export default node;
