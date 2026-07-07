/**
 * @generated SignedSource<<40c01c4ee38da072c840ef58f9bf6aab>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type JobStatus = "canceled" | "canceling" | "failed" | "finished" | "pending" | "queued" | "running" | "%future added value";
export type RunnerType = "group" | "shared" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type RunJobDialog_jobs$data = ReadonlyArray<{
  readonly id: string;
  readonly metadata: {
    readonly createdAt: any;
  };
  readonly runner: {
    readonly groupPath: string;
    readonly id: string;
    readonly name: string;
    readonly type: RunnerType;
  } | null | undefined;
  readonly runnerPath: string | null | undefined;
  readonly status: JobStatus;
  readonly tags: ReadonlyArray<string>;
  readonly timestamps: {
    readonly finishedAt: any | null | undefined;
    readonly pendingAt: any | null | undefined;
    readonly runningAt: any | null | undefined;
  };
  readonly " $fragmentType": "RunJobDialog_jobs";
}>;
export type RunJobDialog_jobs$key = ReadonlyArray<{
  readonly " $data"?: RunJobDialog_jobs$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunJobDialog_jobs">;
}>;

const node: ReaderFragment = (function(){
var v0 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
};
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": {
    "plural": true
  },
  "name": "RunJobDialog_jobs",
  "selections": [
    (v0/*: any*/),
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
      "name": "tags",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "Runner",
      "kind": "LinkedField",
      "name": "runner",
      "plural": false,
      "selections": [
        (v0/*: any*/),
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
          "name": "type",
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
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "runnerPath",
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
          "name": "pendingAt",
          "storageKey": null
        },
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
})();

(node as any).hash = "d5dfc3c08df29737bfbdf3862fa17d2b";

export default node;
