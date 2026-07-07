/**
 * @generated SignedSource<<683630b31b2a26aabcf1c62a9d389009>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type RunJobDialogApplyQuery$variables = {
  after?: string | null | undefined;
  first: number;
  id: string;
};
export type RunJobDialogApplyQuery$data = {
  readonly node: {
    readonly id?: string;
    readonly " $fragmentSpreads": FragmentRefs<"RunJobDialogApplyFragment_jobs">;
  } | null | undefined;
};
export type RunJobDialogApplyQuery = {
  response: RunJobDialogApplyQuery$data;
  variables: RunJobDialogApplyQuery$variables;
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
  "name": "id"
},
v3 = [
  {
    "kind": "Variable",
    "name": "id",
    "variableName": "id"
  }
],
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "__typename",
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
    "kind": "Literal",
    "name": "sort",
    "value": "CREATED_AT_DESC"
  }
];
return {
  "fragment": {
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/),
      (v2/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "RunJobDialogApplyQuery",
    "selections": [
      {
        "alias": null,
        "args": (v3/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          {
            "kind": "InlineFragment",
            "selections": [
              (v4/*: any*/),
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "RunJobDialogApplyFragment_jobs"
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
    "argumentDefinitions": [
      (v2/*: any*/),
      (v1/*: any*/),
      (v0/*: any*/)
    ],
    "kind": "Operation",
    "name": "RunJobDialogApplyQuery",
    "selections": [
      {
        "alias": null,
        "args": (v3/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          (v5/*: any*/),
          (v4/*: any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              {
                "alias": null,
                "args": null,
                "concreteType": "Apply",
                "kind": "LinkedField",
                "name": "apply",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": (v6/*: any*/),
                    "concreteType": "JobConnection",
                    "kind": "LinkedField",
                    "name": "jobs",
                    "plural": false,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "JobEdge",
                        "kind": "LinkedField",
                        "name": "edges",
                        "plural": true,
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": "Job",
                            "kind": "LinkedField",
                            "name": "node",
                            "plural": false,
                            "selections": [
                              (v4/*: any*/),
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
                                "name": "tags",
                                "storageKey": null
                              },
                              {
                                "alias": null,
                                "args": null,
                                "concreteType": "Runner",
                                "kind": "LinkedField",
                                "name": "runner",
                                "plural": false,
                                "selections": [
                                  (v4/*: any*/),
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
                                    "name": "type",
                                    "storageKey": null
                                  },
                                  {
                                    "alias": null,
                                    "args": null,
                                    "kind": "ScalarField",
                                    "name": "groupPath",
                                    "storageKey": null
                                  }
                                ],
                                "storageKey": null
                              },
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
                                "concreteType": "JobTimestamps",
                                "kind": "LinkedField",
                                "name": "timestamps",
                                "plural": false,
                                "selections": [
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
                              (v5/*: any*/)
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
                      "sort"
                    ],
                    "handle": "connection",
                    "key": "RunJobDialogApply_jobs",
                    "kind": "LinkedHandle",
                    "name": "jobs"
                  },
                  (v4/*: any*/)
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
    "cacheID": "dbd1e1a123fe9e21a636d2bb3c16ace5",
    "id": null,
    "metadata": {},
    "name": "RunJobDialogApplyQuery",
    "operationKind": "query",
    "text": "query RunJobDialogApplyQuery(\n  $id: String!\n  $first: Int!\n  $after: String\n) {\n  node(id: $id) {\n    __typename\n    ... on Run {\n      id\n      ...RunJobDialogApplyFragment_jobs\n    }\n    id\n  }\n}\n\nfragment RunJobDialogApplyFragment_jobs on Run {\n  apply {\n    jobs(first: $first, after: $after, sort: CREATED_AT_DESC) {\n      edges {\n        node {\n          id\n          ...RunJobDialog_jobs\n          __typename\n        }\n        cursor\n      }\n      pageInfo {\n        endCursor\n        hasNextPage\n      }\n    }\n    id\n  }\n  id\n}\n\nfragment RunJobDialog_jobs on Job {\n  id\n  status\n  tags\n  runner {\n    id\n    name\n    type\n    groupPath\n  }\n  runnerPath\n  metadata {\n    createdAt\n  }\n  timestamps {\n    pendingAt\n    runningAt\n    finishedAt\n  }\n}\n"
  }
};
})();

(node as any).hash = "91551048c298d21131142d9f286ecace";

export default node;
