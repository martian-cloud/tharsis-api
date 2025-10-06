/**
 * @generated SignedSource<<875ffc9a9ba4c2d98410dfa4d7636adb>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type RunDetailsPlanDiffViewerQuery$variables = {
  id: string;
};
export type RunDetailsPlanDiffViewerQuery$data = {
  readonly node: {
    readonly id?: string;
    readonly " $fragmentSpreads": FragmentRefs<"RunDetailsPlanDiffViewerFragment_run">;
  } | null | undefined;
};
export type RunDetailsPlanDiffViewerQuery = {
  response: RunDetailsPlanDiffViewerQuery$data;
  variables: RunDetailsPlanDiffViewerQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "id"
  }
],
v1 = [
  {
    "kind": "Variable",
    "name": "id",
    "variableName": "id"
  }
],
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "action",
  "storageKey": null
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "unifiedDiff",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "originalSource",
  "storageKey": null
},
v6 = {
  "alias": null,
  "args": null,
  "concreteType": "PlanChangeWarning",
  "kind": "LinkedField",
  "name": "warnings",
  "plural": true,
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "line",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "message",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "changeType",
      "storageKey": null
    }
  ],
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "RunDetailsPlanDiffViewerQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          {
            "kind": "InlineFragment",
            "selections": [
              (v2/*: any*/),
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "RunDetailsPlanDiffViewerFragment_run"
              }
            ],
            "type": "Run",
            "abstractKey": null
          }
        ],
        "storageKey": null
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "RunDetailsPlanDiffViewerQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "__typename",
            "storageKey": null
          },
          (v2/*: any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              {
                "alias": null,
                "args": null,
                "concreteType": "Plan",
                "kind": "LinkedField",
                "name": "plan",
                "plural": false,
                "selections": [
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
                    "concreteType": "PlanChanges",
                    "kind": "LinkedField",
                    "name": "changes",
                    "plural": false,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "PlanResourceChange",
                        "kind": "LinkedField",
                        "name": "resources",
                        "plural": true,
                        "selections": [
                          (v3/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "address",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "providerName",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "resourceType",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "resourceName",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "moduleAddress",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "mode",
                            "storageKey": null
                          },
                          (v4/*: any*/),
                          (v5/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "drifted",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "imported",
                            "storageKey": null
                          },
                          (v6/*: any*/)
                        ],
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "PlanOutputChange",
                        "kind": "LinkedField",
                        "name": "outputs",
                        "plural": true,
                        "selections": [
                          (v3/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "outputName",
                            "storageKey": null
                          },
                          (v4/*: any*/),
                          (v5/*: any*/),
                          (v6/*: any*/)
                        ],
                        "storageKey": null
                      }
                    ],
                    "storageKey": null
                  },
                  (v2/*: any*/)
                ],
                "storageKey": null
              }
            ],
            "type": "Run",
            "abstractKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "a4bec6157835863db327658a82df464a",
    "id": null,
    "metadata": {},
    "name": "RunDetailsPlanDiffViewerQuery",
    "operationKind": "query",
    "text": "query RunDetailsPlanDiffViewerQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ... on Run {\n      id\n      ...RunDetailsPlanDiffViewerFragment_run\n    }\n    id\n  }\n}\n\nfragment RunDetailsPlanDiffViewerFragment_run on Run {\n  plan {\n    status\n    changes {\n      resources {\n        action\n        address\n        providerName\n        resourceType\n        resourceName\n        moduleAddress\n        mode\n        unifiedDiff\n        originalSource\n        drifted\n        imported\n        warnings {\n          line\n          message\n          changeType\n        }\n      }\n      outputs {\n        action\n        outputName\n        unifiedDiff\n        originalSource\n        warnings {\n          line\n          message\n          changeType\n        }\n      }\n    }\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "6c5aa2a8c3ab44a15a83b6414f867b4b";

export default node;
