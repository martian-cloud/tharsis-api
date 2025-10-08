/**
 * @generated SignedSource<<a30f1f6ead28cab534ed34365fb9de0c>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type ApplyStatus = "canceled" | "created" | "errored" | "finished" | "pending" | "queued" | "running" | "%future added value";
export type PlanStatus = "canceled" | "errored" | "finished" | "pending" | "queued" | "running" | "%future added value";
export type RunStatus = "applied" | "apply_queued" | "applying" | "canceled" | "errored" | "pending" | "plan_queued" | "planned" | "planned_and_finished" | "planning" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type RunDetailsSidebarFragment_details$data = {
  readonly apply: {
    readonly currentJob: {
      readonly cancelRequested: boolean;
      readonly runnerPath: string | null | undefined;
    } | null | undefined;
    readonly metadata: {
      readonly createdAt: any;
    };
    readonly status: ApplyStatus;
  } | null | undefined;
  readonly assessment: boolean;
  readonly configurationVersion: {
    readonly id: string;
  } | null | undefined;
  readonly createdBy: string;
  readonly id: string;
  readonly isDestroy: boolean;
  readonly metadata: {
    readonly createdAt: any;
  };
  readonly moduleSource: string | null | undefined;
  readonly moduleVersion: string | null | undefined;
  readonly plan: {
    readonly currentJob: {
      readonly cancelRequested: boolean;
      readonly runnerPath: string | null | undefined;
    } | null | undefined;
    readonly metadata: {
      readonly createdAt: any;
    };
    readonly status: PlanStatus;
  };
  readonly status: RunStatus;
  readonly workspace: {
    readonly fullPath: string;
  };
  readonly " $fragmentType": "RunDetailsSidebarFragment_details";
};
export type RunDetailsSidebarFragment_details$key = {
  readonly " $data"?: RunDetailsSidebarFragment_details$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunDetailsSidebarFragment_details">;
};

const node: ReaderFragment = (function(){
var v0 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v1 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "status",
  "storageKey": null
},
v2 = {
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
v3 = [
  (v1/*: any*/),
  (v2/*: any*/),
  {
    "alias": null,
    "args": null,
    "concreteType": "Job",
    "kind": "LinkedField",
    "name": "currentJob",
    "plural": false,
    "selections": [
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
        "kind": "ScalarField",
        "name": "cancelRequested",
        "storageKey": null
      }
    ],
    "storageKey": null
  }
];
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "RunDetailsSidebarFragment_details",
  "selections": [
    (v0/*: any*/),
    (v1/*: any*/),
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
      "kind": "ScalarField",
      "name": "isDestroy",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "assessment",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "moduleSource",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "moduleVersion",
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
    },
    (v2/*: any*/),
    {
      "alias": null,
      "args": null,
      "concreteType": "ConfigurationVersion",
      "kind": "LinkedField",
      "name": "configurationVersion",
      "plural": false,
      "selections": [
        (v0/*: any*/)
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
      "selections": (v3/*: any*/),
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "Apply",
      "kind": "LinkedField",
      "name": "apply",
      "plural": false,
      "selections": (v3/*: any*/),
      "storageKey": null
    }
  ],
  "type": "Run",
  "abstractKey": null
};
})();

(node as any).hash = "521c1437da1f7f14b2c80fc37e209c14";

export default node;
