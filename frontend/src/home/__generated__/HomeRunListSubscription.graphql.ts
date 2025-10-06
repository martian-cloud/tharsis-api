/**
 * @generated SignedSource<<93d7ef1bbcc559726049aa3526931fc1>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type RunSubscriptionInput = {
  runId?: string | null | undefined;
  workspaceId?: string | null | undefined;
  workspacePath?: string | null | undefined;
};
export type HomeRunListSubscription$variables = {
  input: RunSubscriptionInput;
};
export type HomeRunListSubscription$data = {
  readonly workspaceRunEvents: {
    readonly action: string;
    readonly run: {
      readonly id: string;
      readonly " $fragmentSpreads": FragmentRefs<"HomeRunListItemFragment_run">;
    };
  };
};
export type HomeRunListSubscription = {
  response: HomeRunListSubscription$data;
  variables: HomeRunListSubscription$variables;
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
},
v4 = [
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "status",
    "storageKey": null
  },
  (v3/*: any*/)
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "HomeRunListSubscription",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "RunEvent",
        "kind": "LinkedField",
        "name": "workspaceRunEvents",
        "plural": false,
        "selections": [
          (v2/*: any*/),
          {
            "alias": null,
            "args": null,
            "concreteType": "Run",
            "kind": "LinkedField",
            "name": "run",
            "plural": false,
            "selections": [
              (v3/*: any*/),
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "HomeRunListItemFragment_run"
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
    "name": "HomeRunListSubscription",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "RunEvent",
        "kind": "LinkedField",
        "name": "workspaceRunEvents",
        "plural": false,
        "selections": [
          (v2/*: any*/),
          {
            "alias": null,
            "args": null,
            "concreteType": "Run",
            "kind": "LinkedField",
            "name": "run",
            "plural": false,
            "selections": [
              (v3/*: any*/),
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
                "selections": (v4/*: any*/),
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "Apply",
                "kind": "LinkedField",
                "name": "apply",
                "plural": false,
                "selections": (v4/*: any*/),
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
                  },
                  (v3/*: any*/)
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
    "cacheID": "a7f22c598a8bbc927e90516ccc3f7e0a",
    "id": null,
    "metadata": {},
    "name": "HomeRunListSubscription",
    "operationKind": "subscription",
    "text": "subscription HomeRunListSubscription(\n  $input: RunSubscriptionInput!\n) {\n  workspaceRunEvents(input: $input) {\n    action\n    run {\n      id\n      ...HomeRunListItemFragment_run\n    }\n  }\n}\n\nfragment HomeRunListItemFragment_run on Run {\n  id\n  createdBy\n  metadata {\n    createdAt\n  }\n  plan {\n    status\n    id\n  }\n  apply {\n    status\n    id\n  }\n  workspace {\n    fullPath\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "99506e643a552642c2b1c813c03d36e0";

export default node;
