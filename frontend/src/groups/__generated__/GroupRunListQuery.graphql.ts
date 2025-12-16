/**
 * @generated SignedSource<<04d0201f0af587392b44ddea293ba889>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupRunListQuery$variables = {
  after?: string | null | undefined;
  first?: number | null | undefined;
  groupPath: string;
  includeNestedRuns?: boolean | null | undefined;
  workspaceAssessment?: boolean | null | undefined;
};
export type GroupRunListQuery$data = {
  readonly " $fragmentSpreads": FragmentRefs<"GroupRunListFragment_group">;
};
export type GroupRunListQuery = {
  response: GroupRunListQuery$data;
  variables: GroupRunListQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "after"
},
v1 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "first"
},
v2 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "groupPath"
},
v3 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "includeNestedRuns"
},
v4 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "workspaceAssessment"
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v6 = [
  {
    "kind": "Variable",
    "name": "after",
    "variableName": "after"
  },
  {
    "kind": "Variable",
    "name": "first",
    "variableName": "first"
  },
  {
    "kind": "Variable",
    "name": "includeNestedRuns",
    "variableName": "includeNestedRuns"
  },
  {
    "kind": "Literal",
    "name": "sort",
    "value": "CREATED_AT_DESC"
  },
  {
    "kind": "Variable",
    "name": "workspaceAssessment",
    "variableName": "workspaceAssessment"
  }
],
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "status",
  "storageKey": null
},
v8 = [
  (v7/*: any*/),
  (v5/*: any*/)
];
return {
  "fragment": {
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/),
      (v2/*: any*/),
      (v3/*: any*/),
      (v4/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "GroupRunListQuery",
    "selections": [
      {
        "args": null,
        "kind": "FragmentSpread",
        "name": "GroupRunListFragment_group"
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [
      (v1/*: any*/),
      (v0/*: any*/),
      (v2/*: any*/),
      (v4/*: any*/),
      (v3/*: any*/)
    ],
    "kind": "Operation",
    "name": "GroupRunListQuery",
    "selections": [
      {
        "alias": null,
        "args": [
          {
            "kind": "Variable",
            "name": "fullPath",
            "variableName": "groupPath"
          }
        ],
        "concreteType": "Group",
        "kind": "LinkedField",
        "name": "group",
        "plural": false,
        "selections": [
          (v5/*: any*/),
          {
            "alias": null,
            "args": (v6/*: any*/),
            "concreteType": "RunConnection",
            "kind": "LinkedField",
            "name": "runs",
            "plural": false,
            "selections": [
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "totalCount",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "RunEdge",
                "kind": "LinkedField",
                "name": "edges",
                "plural": true,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "Run",
                    "kind": "LinkedField",
                    "name": "node",
                    "plural": false,
                    "selections": [
                      (v5/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "__typename",
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
                            "name": "trn",
                            "storageKey": null
                          }
                        ],
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "createdBy",
                        "storageKey": null
                      },
                      (v7/*: any*/),
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
                          (v5/*: any*/)
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
                        "selections": (v8/*: any*/),
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "Apply",
                        "kind": "LinkedField",
                        "name": "apply",
                        "plural": false,
                        "selections": (v8/*: any*/),
                        "storageKey": null
                      }
                    ],
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "cursor",
                    "storageKey": null
                  }
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "PageInfo",
                "kind": "LinkedField",
                "name": "pageInfo",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "endCursor",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "hasNextPage",
                    "storageKey": null
                  }
                ],
                "storageKey": null
              }
            ],
            "storageKey": null
          },
          {
            "alias": null,
            "args": (v6/*: any*/),
            "filters": [
              "sort",
              "workspaceAssessment",
              "includeNestedRuns"
            ],
            "handle": "connection",
            "key": "GroupRunList_runs",
            "kind": "LinkedHandle",
            "name": "runs"
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "387eeaf1a10774de3e13d86af82fe439",
    "id": null,
    "metadata": {},
    "name": "GroupRunListQuery",
    "operationKind": "query",
    "text": "query GroupRunListQuery(\n  $first: Int\n  $after: String\n  $groupPath: String!\n  $workspaceAssessment: Boolean\n  $includeNestedRuns: Boolean\n) {\n  ...GroupRunListFragment_group\n}\n\nfragment GroupRunListFragment_group on Query {\n  group(fullPath: $groupPath) {\n    id\n    runs(first: $first, after: $after, sort: CREATED_AT_DESC, workspaceAssessment: $workspaceAssessment, includeNestedRuns: $includeNestedRuns) {\n      totalCount\n      edges {\n        node {\n          id\n          __typename\n        }\n        cursor\n      }\n      ...RunListFragment_runConnection\n      pageInfo {\n        endCursor\n        hasNextPage\n      }\n    }\n  }\n}\n\nfragment RunListFragment_runConnection on RunConnection {\n  totalCount\n  edges {\n    node {\n      id\n      ...RunListItemFragment_run\n    }\n  }\n}\n\nfragment RunListItemFragment_run on Run {\n  metadata {\n    createdAt\n    trn\n  }\n  id\n  createdBy\n  status\n  isDestroy\n  assessment\n  workspace {\n    fullPath\n    id\n  }\n  plan {\n    status\n    id\n  }\n  apply {\n    status\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "1a3c2011a038590985708f737f62ec43";

export default node;
