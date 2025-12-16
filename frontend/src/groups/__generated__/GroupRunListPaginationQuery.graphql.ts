/**
 * @generated SignedSource<<5b7f326a3b1485c8a6b4baf68e42a6fc>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupRunListPaginationQuery$variables = {
  after?: string | null | undefined;
  first?: number | null | undefined;
  groupPath: string;
  includeNestedRuns?: boolean | null | undefined;
  workspaceAssessment?: boolean | null | undefined;
};
export type GroupRunListPaginationQuery$data = {
  readonly " $fragmentSpreads": FragmentRefs<"GroupRunListFragment_group">;
};
export type GroupRunListPaginationQuery = {
  response: GroupRunListPaginationQuery$data;
  variables: GroupRunListPaginationQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "after"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "first"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "groupPath"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "includeNestedRuns"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "workspaceAssessment"
  }
],
v1 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v2 = [
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
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "status",
  "storageKey": null
},
v4 = [
  (v3/*: any*/),
  (v1/*: any*/)
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "GroupRunListPaginationQuery",
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
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "GroupRunListPaginationQuery",
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
          (v1/*: any*/),
          {
            "alias": null,
            "args": (v2/*: any*/),
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
                      (v1/*: any*/),
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
                      (v3/*: any*/),
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
                          (v1/*: any*/)
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
            "args": (v2/*: any*/),
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
    "cacheID": "8781563049dcc07aacfff2ee6f1a4639",
    "id": null,
    "metadata": {},
    "name": "GroupRunListPaginationQuery",
    "operationKind": "query",
    "text": "query GroupRunListPaginationQuery(\n  $after: String\n  $first: Int\n  $groupPath: String!\n  $includeNestedRuns: Boolean\n  $workspaceAssessment: Boolean\n) {\n  ...GroupRunListFragment_group\n}\n\nfragment GroupRunListFragment_group on Query {\n  group(fullPath: $groupPath) {\n    id\n    runs(first: $first, after: $after, sort: CREATED_AT_DESC, workspaceAssessment: $workspaceAssessment, includeNestedRuns: $includeNestedRuns) {\n      totalCount\n      edges {\n        node {\n          id\n          __typename\n        }\n        cursor\n      }\n      ...RunListFragment_runConnection\n      pageInfo {\n        endCursor\n        hasNextPage\n      }\n    }\n  }\n}\n\nfragment RunListFragment_runConnection on RunConnection {\n  totalCount\n  edges {\n    node {\n      id\n      ...RunListItemFragment_run\n    }\n  }\n}\n\nfragment RunListItemFragment_run on Run {\n  metadata {\n    createdAt\n    trn\n  }\n  id\n  createdBy\n  status\n  isDestroy\n  assessment\n  workspace {\n    fullPath\n    id\n  }\n  plan {\n    status\n    id\n  }\n  apply {\n    status\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "b907b2e0a6b18fb24721d74c9f26681d";

export default node;
