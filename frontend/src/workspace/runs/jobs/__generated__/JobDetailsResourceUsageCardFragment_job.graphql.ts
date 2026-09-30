/**
 * @generated SignedSource<<e984df50885afcfda15f8965d35eea5f>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type JobDetailsResourceUsageCardFragment_job$data = {
  readonly resourceUsageMetrics: {
    readonly totalCpuTimeMs: number | null | undefined;
  } | null | undefined;
  readonly " $fragmentSpreads": FragmentRefs<"JobResourceUsageMetricsFragment_job">;
  readonly " $fragmentType": "JobDetailsResourceUsageCardFragment_job";
};
export type JobDetailsResourceUsageCardFragment_job$key = {
  readonly " $data"?: JobDetailsResourceUsageCardFragment_job$data;
  readonly " $fragmentSpreads": FragmentRefs<"JobDetailsResourceUsageCardFragment_job">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "JobDetailsResourceUsageCardFragment_job",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "JobResourceUsageMetrics",
      "kind": "LinkedField",
      "name": "resourceUsageMetrics",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "totalCpuTimeMs",
          "storageKey": null
        }
      ],
      "storageKey": null
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "JobResourceUsageMetricsFragment_job"
    }
  ],
  "type": "Job",
  "abstractKey": null
};

(node as any).hash = "2ff0541196e93e5af57a015414f2aaf9";

export default node;
