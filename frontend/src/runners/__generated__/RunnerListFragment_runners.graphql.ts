/**
 * @generated SignedSource<<edb4ab426729c280affe2f12a52c113a>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type RunnerListFragment_runners$data = {
  readonly edges: ReadonlyArray<{
    readonly node: {
      readonly groupPath: string;
      readonly id: string;
      readonly " $fragmentSpreads": FragmentRefs<"RunnerListItemFragment_runner">;
    } | null | undefined;
  } | null | undefined> | null | undefined;
  readonly " $fragmentType": "RunnerListFragment_runners";
};
export type RunnerListFragment_runners$key = {
  readonly " $data"?: RunnerListFragment_runners$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunnerListFragment_runners">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "RunnerListFragment_runners",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "RunnerEdge",
      "kind": "LinkedField",
      "name": "edges",
      "plural": true,
      "selections": [
        {
          "alias": null,
          "args": null,
          "concreteType": "Runner",
          "kind": "LinkedField",
          "name": "node",
          "plural": false,
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
              "name": "groupPath",
              "storageKey": null
            },
            {
              "args": null,
              "kind": "FragmentSpread",
              "name": "RunnerListItemFragment_runner"
            }
          ],
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "RunnerConnection",
  "abstractKey": null
};

(node as any).hash = "7a3f5808f50da2b81f272d1f1519abec";

export default node;
