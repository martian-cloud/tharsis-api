/**
 * @generated SignedSource<<90ac313b83cd38888f16bf326658f2b1>>
 * @lightSyntaxTransform
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
    readonly run: {
      readonly id: string;
      readonly " $fragmentSpreads": FragmentRefs<"RunDetailsPlanDiffViewerFragment_run">;
    } | null | undefined;
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
    "argumentDefinitions": (v0/*:: as any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "RunDetailsPlanDiffViewerQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*:: as any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          {
            "fragment": {
              "kind": "InlineFragment",
              "selections": [
                (v2/*:: as any*/),
                {
                  "args": null,
                  "kind": "FragmentSpread",
                  "name": "RunDetailsPlanDiffViewerFragment_run"
                }
              ],
              "type": "Run",
              "abstractKey": null
            },
            "kind": "AliasedInlineFragmentSpread",
            "name": "run"
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
    "argumentDefinitions": (v0/*:: as any*/),
    "kind": "Operation",
    "name": "RunDetailsPlanDiffViewerQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*:: as any*/),
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
          (v2/*:: as any*/),
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
                          (v3/*:: as any*/),
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
                          (v4/*:: as any*/),
                          (v5/*:: as any*/),
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
                          (v6/*:: as any*/)
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
                          (v3/*:: as any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "outputName",
                            "storageKey": null
                          },
                          (v4/*:: as any*/),
                          (v5/*:: as any*/),
                          (v6/*:: as any*/)
                        ],
                        "storageKey": null
                      }
                    ],
                    "storageKey": null
                  },
                  (v2/*:: as any*/)
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

(node as any).hash = "c41153f03e90250aa70851c41b528db1";

export default node;
