/**
 * @generated SignedSource<<6d659ed913313f34c5a4163da7bfb67e>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type JobStatus = "canceled" | "canceling" | "failed" | "finished" | "pending" | "queued" | "running" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type RunTaskStagePolicyCheckPreviousJobsMenu_job$data = {
  readonly id: string;
  readonly metadata: {
    readonly createdAt: any;
  };
  readonly status: JobStatus;
  readonly timestamps: {
    readonly finishedAt: any | null | undefined;
    readonly runningAt: any | null | undefined;
  };
  readonly " $fragmentType": "RunTaskStagePolicyCheckPreviousJobsMenu_job";
};
export type RunTaskStagePolicyCheckPreviousJobsMenu_job$key = {
  readonly " $data"?: RunTaskStagePolicyCheckPreviousJobsMenu_job$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunTaskStagePolicyCheckPreviousJobsMenu_job">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "RunTaskStagePolicyCheckPreviousJobsMenu_job",
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
      "concreteType": "JobTimestamps",
      "kind": "LinkedField",
      "name": "timestamps",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "runningAt",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "finishedAt",
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Job",
  "abstractKey": null
};

(node as any).hash = "0696e66490f3eb9de65baa837443c1c2";

export default node;
