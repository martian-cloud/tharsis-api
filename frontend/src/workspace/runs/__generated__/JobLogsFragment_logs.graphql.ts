/**
 * @generated SignedSource<<4ceea3e0b0c6cb8ceabbca5bafecd0ec>>
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
  "argumentDefinitions": [],
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
          "kind": "Literal",
          "name": "limit",
          "value": 51200
        },
        {
          "kind": "Literal",
          "name": "startOffset",
          "value": 0
        }
      ],
      "kind": "ScalarField",
      "name": "logs",
      "storageKey": "logs(limit:51200,startOffset:0)"
    }
  ],
  "type": "Job",
  "abstractKey": null
};

(node as any).hash = "090e1292e76cd3824800e0949a8c365c";

export default node;
