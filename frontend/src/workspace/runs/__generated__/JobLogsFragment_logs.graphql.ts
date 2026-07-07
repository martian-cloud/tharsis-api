/**
 * @generated SignedSource<<0ba51f0b421788f350aae34dc30576a3>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type JobStatus = "canceled" | "canceling" | "failed" | "finished" | "pending" | "queued" | "running" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type JobLogsFragment_logs$data = {
  readonly completed: boolean;
  readonly id: string;
  readonly logLastUpdatedAt: any | null | undefined;
  readonly logSize: number;
  readonly logs: string;
  readonly status: JobStatus;
  readonly " $fragmentType": "JobLogsFragment_logs";
};
export type JobLogsFragment_logs$key = {
  readonly " $data"?: JobLogsFragment_logs$data;
  readonly " $fragmentSpreads": FragmentRefs<"JobLogsFragment_logs">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [
    {
      "kind": "RootArgument",
      "name": "limit"
    },
    {
      "kind": "RootArgument",
      "name": "startOffset"
    }
  ],
  "kind": "Fragment",
  "metadata": null,
  "name": "JobLogsFragment_logs",
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
      "name": "status",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "completed",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "logLastUpdatedAt",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "logSize",
      "storageKey": null
    },
    {
      "alias": null,
      "args": [
        {
          "kind": "Variable",
          "name": "limit",
          "variableName": "limit"
        },
        {
          "kind": "Variable",
          "name": "startOffset",
          "variableName": "startOffset"
        }
      ],
      "kind": "ScalarField",
      "name": "logs",
      "storageKey": null
    }
  ],
  "type": "Job",
  "abstractKey": null
};

(node as any).hash = "756a5cfe3456820b806dceff6c6260f8";

export default node;
