/**
 * @generated SignedSource<<1c6114143545cfaca56ec324ae36c862>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type JobResourceUsageMetricsFragment_job$data = {
  readonly resourceUsageLimits: {
    readonly diskReadBytes: number | null | undefined;
    readonly diskWriteBytes: number | null | undefined;
    readonly memoryBytes: number | null | undefined;
    readonly networkReceivedBytes: number | null | undefined;
    readonly networkSentBytes: number | null | undefined;
  } | null | undefined;
  readonly resourceUsageMetrics: {
    readonly peakMemoryBytes: number | null | undefined;
    readonly totalCpuTimeMs: number | null | undefined;
    readonly totalDiskReadBytes: number | null | undefined;
    readonly totalDiskWriteBytes: number | null | undefined;
    readonly totalNetworkReceivedBytes: number | null | undefined;
    readonly totalNetworkSentBytes: number | null | undefined;
  } | null | undefined;
  readonly " $fragmentType": "JobResourceUsageMetricsFragment_job";
};
export type JobResourceUsageMetricsFragment_job$key = {
  readonly " $data"?: JobResourceUsageMetricsFragment_job$data;
  readonly " $fragmentSpreads": FragmentRefs<"JobResourceUsageMetricsFragment_job">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "JobResourceUsageMetricsFragment_job",
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
          "name": "peakMemoryBytes",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "totalCpuTimeMs",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "totalNetworkReceivedBytes",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "totalNetworkSentBytes",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "totalDiskReadBytes",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "totalDiskWriteBytes",
          "storageKey": null
        }
      ],
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "JobResourceUsageLimits",
      "kind": "LinkedField",
      "name": "resourceUsageLimits",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "memoryBytes",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "networkReceivedBytes",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "networkSentBytes",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "diskReadBytes",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "diskWriteBytes",
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Job",
  "abstractKey": null
};

(node as any).hash = "676970a230b6c906351b6dfef3855541";

export default node;
