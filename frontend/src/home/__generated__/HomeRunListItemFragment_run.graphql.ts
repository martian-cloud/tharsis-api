/**
 * @generated SignedSource<<b02f62bfe43e3c47969fb38de29a0979>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type ApplyStatus = "canceled" | "created" | "errored" | "finished" | "pending" | "queued" | "running" | "%future added value";
export type PlanStatus = "canceled" | "errored" | "finished" | "pending" | "queued" | "running" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type HomeRunListItemFragment_run$data = {
  readonly apply: {
    readonly status: ApplyStatus;
  } | null | undefined;
  readonly createdBy: string;
  readonly id: string;
  readonly metadata: {
    readonly createdAt: any;
  };
  readonly plan: {
    readonly status: PlanStatus;
  };
  readonly workspace: {
    readonly fullPath: string;
  };
  readonly " $fragmentType": "HomeRunListItemFragment_run";
};
export type HomeRunListItemFragment_run$key = {
  readonly " $data"?: HomeRunListItemFragment_run$data;
  readonly " $fragmentSpreads": FragmentRefs<"HomeRunListItemFragment_run">;
};

const node: ReaderFragment = (function(){
var v0 = [
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "status",
    "storageKey": null
  }
];
return {
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
      "kind": "ScalarField",
      "name": "createdBy",
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
      "concreteType": "Plan",
      "kind": "LinkedField",
      "name": "plan",
      "plural": false,
      "selections": (v0/*: any*/),
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "Apply",
      "kind": "LinkedField",
      "name": "apply",
      "plural": false,
      "selections": (v0/*: any*/),
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
    }
  ],
  "type": "Run",
  "abstractKey": null
};
})();

(node as any).hash = "0672250ea469c286561cdbbf114e1691";

export default node;
