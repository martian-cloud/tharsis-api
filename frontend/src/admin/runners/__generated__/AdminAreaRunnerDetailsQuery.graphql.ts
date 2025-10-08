/**
 * @generated SignedSource<<875d83f7e40a1b959c4781200ef66c23>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type AdminAreaRunnerDetailsQuery$variables = {
  id: string;
};
export type AdminAreaRunnerDetailsQuery$data = {
  readonly node: {
    readonly id?: string;
    readonly name?: string;
    readonly " $fragmentSpreads": FragmentRefs<"RunnerDetailsFragment_runner">;
  } | null | undefined;
};
export type AdminAreaRunnerDetailsQuery = {
  response: AdminAreaRunnerDetailsQuery$data;
  variables: AdminAreaRunnerDetailsQuery$variables;
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
  "name": "name",
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "AdminAreaRunnerDetailsQuery",
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
              (v3/*: any*/),
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "RunnerDetailsFragment_runner"
              }
            ],
            "type": "Runner",
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
    "name": "AdminAreaRunnerDetailsQuery",
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
              (v3/*: any*/),
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
                "kind": "ScalarField",
                "name": "disabled",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "description",
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
                "kind": "ScalarField",
                "name": "tags",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "runUntaggedJobs",
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
                "args": [
                  {
                    "kind": "Literal",
                    "name": "first",
                    "value": 0
                  }
                ],
                "concreteType": "ServiceAccountConnection",
                "kind": "LinkedField",
                "name": "assignedServiceAccounts",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "totalCount",
                    "storageKey": null
                  }
                ],
                "storageKey": "assignedServiceAccounts(first:0)"
              },
              {
                "alias": null,
                "args": [
                  {
                    "kind": "Literal",
                    "name": "first",
                    "value": 1
                  },
                  {
                    "kind": "Literal",
                    "name": "sort",
                    "value": "LAST_CONTACTED_AT_DESC"
                  }
                ],
                "concreteType": "RunnerSessionConnection",
                "kind": "LinkedField",
                "name": "sessions",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "RunnerSessionEdge",
                    "kind": "LinkedField",
                    "name": "edges",
                    "plural": true,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "RunnerSession",
                        "kind": "LinkedField",
                        "name": "node",
                        "plural": false,
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "active",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "lastContacted",
                            "storageKey": null
                          },
                          (v2/*: any*/)
                        ],
                        "storageKey": null
                      }
                    ],
                    "storageKey": null
                  }
                ],
                "storageKey": "sessions(first:1,sort:\"LAST_CONTACTED_AT_DESC\")"
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "resourcePath",
                "storageKey": null
              }
            ],
            "type": "Runner",
            "abstractKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "158bce26e814d93d1cca447b59085972",
    "id": null,
    "metadata": {},
    "name": "AdminAreaRunnerDetailsQuery",
    "operationKind": "query",
    "text": "query AdminAreaRunnerDetailsQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ... on Runner {\n      id\n      name\n      ...RunnerDetailsFragment_runner\n    }\n    id\n  }\n}\n\nfragment AssignedServiceAccountListFragment_runner on Runner {\n  id\n  resourcePath\n}\n\nfragment RunnerDetailsFragment_runner on Runner {\n  id\n  name\n  type\n  disabled\n  description\n  createdBy\n  tags\n  runUntaggedJobs\n  metadata {\n    createdAt\n    trn\n  }\n  assignedServiceAccounts(first: 0) {\n    totalCount\n  }\n  sessions(first: 1, sort: LAST_CONTACTED_AT_DESC) {\n    edges {\n      node {\n        active\n        lastContacted\n        id\n      }\n    }\n  }\n  ...AssignedServiceAccountListFragment_runner\n}\n"
  }
};
})();

(node as any).hash = "02f03e012ba34fb905d49e22004faa71";

export default node;
