/**
 * @generated SignedSource<<f5077b94638db90f146292fbd86df0fc>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type JobSubscriptionInput = {
  runnerId?: string | null | undefined;
  workspaceId?: string | null | undefined;
};
export type RunnerJobListEventsSubscription$variables = {
  input: JobSubscriptionInput;
};
export type RunnerJobListEventsSubscription$data = {
  readonly jobEvents: {
    readonly action: string;
    readonly job: {
      readonly id: string;
      readonly " $fragmentSpreads": FragmentRefs<"RunnerJobListItemFragment">;
    };
  };
};
export type RunnerJobListEventsSubscription = {
  response: RunnerJobListEventsSubscription$data;
  variables: RunnerJobListEventsSubscription$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "input"
  }
],
v1 = [
  {
    "kind": "Variable",
    "name": "input",
    "variableName": "input"
  }
],
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "action",
  "storageKey": null
},
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "RunnerJobListEventsSubscription",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "JobEvent",
        "kind": "LinkedField",
        "name": "jobEvents",
        "plural": false,
        "selections": [
          (v2/*: any*/),
          {
            "alias": null,
            "args": null,
            "concreteType": "Job",
            "kind": "LinkedField",
            "name": "job",
            "plural": false,
            "selections": [
              (v3/*: any*/),
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "RunnerJobListItemFragment"
              }
            ],
            "storageKey": null
          }
        ],
        "storageKey": null
      }
    ],
    "type": "Subscription",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "RunnerJobListEventsSubscription",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "JobEvent",
        "kind": "LinkedField",
        "name": "jobEvents",
        "plural": false,
        "selections": [
          (v2/*: any*/),
          {
            "alias": null,
            "args": null,
            "concreteType": "Job",
            "kind": "LinkedField",
            "name": "job",
            "plural": false,
            "selections": [
              (v3/*: any*/),
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
                "name": "type",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "Run",
                "kind": "LinkedField",
                "name": "run",
                "plural": false,
                "selections": [
                  (v3/*: any*/)
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
                    "name": "queuedAt",
                    "storageKey": null
                  },
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
                    "name": "name",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "fullPath",
                    "storageKey": null
                  },
                  (v3/*: any*/)
                ],
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
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "updatedAt",
                    "storageKey": null
                  }
                ],
                "storageKey": null
              }
            ],
            "storageKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "3c11b15f98c6a34109416eaae09dba6a",
    "id": null,
    "metadata": {},
    "name": "RunnerJobListEventsSubscription",
    "operationKind": "subscription",
    "text": "subscription RunnerJobListEventsSubscription(\n  $input: JobSubscriptionInput!\n) {\n  jobEvents(input: $input) {\n    action\n    job {\n      id\n      ...RunnerJobListItemFragment\n    }\n  }\n}\n\nfragment RunnerJobListItemFragment on Job {\n  id\n  status\n  type\n  run {\n    id\n  }\n  timestamps {\n    queuedAt\n    pendingAt\n    runningAt\n    finishedAt\n  }\n  workspace {\n    name\n    fullPath\n    id\n  }\n  metadata {\n    createdAt\n    updatedAt\n  }\n}\n"
  }
};
})();

(node as any).hash = "7ca3c03e111881764acf6cdd3ca4bd35";

export default node;
