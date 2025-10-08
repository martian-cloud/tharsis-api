/**
 * @generated SignedSource<<5e8e33e0e81d0ccaed77feae41501817>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type RunDetailsPlanSummaryFragment_plan$data = {
  readonly summary: {
    readonly outputAdditions: number;
    readonly outputChanges: number;
    readonly outputDestructions: number;
    readonly resourceAdditions: number;
    readonly resourceChanges: number;
    readonly resourceDestructions: number;
    readonly resourceDrift: number;
    readonly resourceImports: number;
  };
  readonly " $fragmentType": "RunDetailsPlanSummaryFragment_plan";
};
export type RunDetailsPlanSummaryFragment_plan$key = {
  readonly " $data"?: RunDetailsPlanSummaryFragment_plan$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunDetailsPlanSummaryFragment_plan">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "RunDetailsPlanSummaryFragment_plan",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "PlanSummary",
      "kind": "LinkedField",
      "name": "summary",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "resourceAdditions",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "resourceChanges",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "resourceDestructions",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "resourceImports",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "resourceDrift",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "outputAdditions",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "outputChanges",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "outputDestructions",
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Plan",
  "abstractKey": null
};

(node as any).hash = "78f85e645007228844c22cdc8e7f4ccc";

export default node;
