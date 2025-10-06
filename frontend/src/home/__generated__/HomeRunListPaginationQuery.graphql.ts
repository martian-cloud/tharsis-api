/**
 * @generated SignedSource<<c6fca009009132a4f19f8e275d4f50ca>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type HomeRunListPaginationQuery$variables = {
  after?: string | null | undefined;
  first?: number | null | undefined;
};
export type HomeRunListPaginationQuery$data = {
  readonly " $fragmentSpreads": FragmentRefs<"HomeRunListFragment_runs">;
};
export type HomeRunListPaginationQuery = {
  response: HomeRunListPaginationQuery$data;
  variables: HomeRunListPaginationQuery$variables;
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
    "name": "first",
    "variableName": "first"
  },
  {
    "kind": "Literal",
    "name": "sort",
    "value": "UPDATED_AT_DESC"
  },
  {
    "kind": "Literal",
    "name": "workspaceAssessment",
    "value": false
  }
],
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v3 = [
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "status",
    "storageKey": null
  },
  (v2/*: any*/)
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "HomeRunListPaginationQuery",
    "selections": [
      {
        "args": null,
        "kind": "FragmentSpread",
        "name": "HomeRunListFragment_runs"
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "HomeRunListPaginationQuery",
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
                      (v2/*: any*/)
                    ],
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
          "sort",
          "workspaceAssessment"
        ],
        "handle": "connection",
        "key": "HomeRunList_runs",
        "kind": "LinkedHandle",
        "name": "runs"
      }
    ]
  },
  "params": {
    "cacheID": "1a6873eaa9b251e9106ec9c634551043",
    "id": null,
    "metadata": {},
    "name": "HomeRunListPaginationQuery",
    "operationKind": "query",
    "text": "query HomeRunListPaginationQuery(\n  $after: String\n  $first: Int\n) {\n  ...HomeRunListFragment_runs\n}\n\nfragment HomeRunListFragment_runs on Query {\n  runs(first: $first, after: $after, sort: UPDATED_AT_DESC, workspaceAssessment: false) {\n    totalCount\n    edges {\n      node {\n        id\n        ...HomeRunListItemFragment_run\n        __typename\n      }\n      cursor\n    }\n    pageInfo {\n      endCursor\n      hasNextPage\n    }\n  }\n}\n\nfragment HomeRunListItemFragment_run on Run {\n  id\n  createdBy\n  metadata {\n    createdAt\n  }\n  plan {\n    status\n    id\n  }\n  apply {\n    status\n    id\n  }\n  workspace {\n    fullPath\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "c16569127c66edb21ba55ee4130d0d96";

export default node;
