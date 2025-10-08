/**
 * @generated SignedSource<<e66af6e371ed72c8291c39712fabfe8a>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type RunListPaginationQuery$variables = {
  after?: string | null | undefined;
  before?: string | null | undefined;
  first?: number | null | undefined;
  last?: number | null | undefined;
  workspaceAssessment?: boolean | null | undefined;
  workspaceId?: string | null | undefined;
};
export type RunListPaginationQuery$data = {
  readonly " $fragmentSpreads": FragmentRefs<"RunListFragment_runs">;
};
export type RunListPaginationQuery = {
  response: RunListPaginationQuery$data;
  variables: RunListPaginationQuery$variables;
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
    "name": "before"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "first"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "last"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "workspaceAssessment"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "workspaceId"
  }
],
v1 = [
  {
    "kind": "Variable",
    "name": "after",
    "variableName": "after"
  },
  {
    "kind": "Variable",
    "name": "before",
    "variableName": "before"
  },
  {
    "kind": "Variable",
    "name": "first",
    "variableName": "first"
  },
  {
    "kind": "Variable",
    "name": "last",
    "variableName": "last"
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
  },
  {
    "kind": "Variable",
    "name": "workspaceId",
    "variableName": "workspaceId"
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
  "name": "status",
  "storageKey": null
},
v4 = [
  (v3/*: any*/),
  (v2/*: any*/)
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "RunListPaginationQuery",
    "selections": [
      {
        "args": null,
        "kind": "FragmentSpread",
        "name": "RunListFragment_runs"
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "RunListPaginationQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
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
                  (v2/*: any*/),
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
                    "kind": "ScalarField",
                    "name": "__typename",
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
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "hasPreviousPage",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "startCursor",
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
        "args": (v1/*: any*/),
        "filters": [
          "workspaceId",
          "sort",
          "workspaceAssessment"
        ],
        "handle": "connection",
        "key": "RunList_runs",
        "kind": "LinkedHandle",
        "name": "runs"
      }
    ]
  },
  "params": {
    "cacheID": "60fb9a0b6851cdbd8a7f7c01b9c5bcdc",
    "id": null,
    "metadata": {},
    "name": "RunListPaginationQuery",
    "operationKind": "query",
    "text": "query RunListPaginationQuery(\n  $after: String\n  $before: String\n  $first: Int\n  $last: Int\n  $workspaceAssessment: Boolean\n  $workspaceId: String\n) {\n  ...RunListFragment_runs\n}\n\nfragment RunListFragment_runs on Query {\n  runs(after: $after, before: $before, first: $first, last: $last, workspaceId: $workspaceId, sort: CREATED_AT_DESC, workspaceAssessment: $workspaceAssessment) {\n    totalCount\n    edges {\n      node {\n        id\n        ...RunListItemFragment_run\n        __typename\n      }\n      cursor\n    }\n    pageInfo {\n      endCursor\n      hasNextPage\n      hasPreviousPage\n      startCursor\n    }\n  }\n}\n\nfragment RunListItemFragment_run on Run {\n  metadata {\n    createdAt\n    trn\n  }\n  id\n  createdBy\n  status\n  isDestroy\n  assessment\n  plan {\n    status\n    id\n  }\n  apply {\n    status\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "fc2e409ecd0d355ae76a6149d827c3c6";

export default node;
