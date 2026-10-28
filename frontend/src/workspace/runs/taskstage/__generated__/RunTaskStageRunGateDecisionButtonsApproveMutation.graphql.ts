/**
 * @generated SignedSource<<7de8abc9b5ab4ab4b90f93f9fcf844bf>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type RunGateDecision = "APPROVE" | "REJECT" | "%future added value";
export type RunGateStatus = "APPROVED" | "CANCELED" | "OVERRIDDEN" | "PENDING" | "%future added value";
export type ApproveRunGateInput = {
  clientMutationId?: string | null | undefined;
  comment?: string | null | undefined;
  decision: RunGateDecision;
  gateId: string;
};
export type RunTaskStageRunGateDecisionButtonsApproveMutation$variables = {
  connections: ReadonlyArray<string>;
  input: ApproveRunGateInput;
};
export type RunTaskStageRunGateDecisionButtonsApproveMutation$data = {
  readonly approveRunGate: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly runGate: {
      readonly approvals: ReadonlyArray<{
        readonly comment: string;
        readonly coveredRules: ReadonlyArray<string>;
        readonly createdBy: string;
        readonly createdByActor: {
          readonly " $fragmentSpreads": FragmentRefs<"ActorDisplayFragment_actor">;
        } | null | undefined;
        readonly decision: RunGateDecision;
        readonly id: string;
        readonly metadata: {
          readonly createdAt: any;
        };
        readonly serviceAccount: {
          readonly id: string;
          readonly name: string;
          readonly resourcePath: string;
        } | null | undefined;
        readonly user: {
          readonly email: string;
          readonly id: string;
          readonly username: string;
        } | null | undefined;
      }>;
      readonly id: string;
      readonly status: RunGateStatus;
    } | null | undefined;
  };
};
export type RunTaskStageRunGateDecisionButtonsApproveMutation = {
  response: RunTaskStageRunGateDecisionButtonsApproveMutation$data;
  variables: RunTaskStageRunGateDecisionButtonsApproveMutation$variables;
};

const node: ConcreteRequest = (function(){
var v0 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "connections"
},
v1 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "input"
},
v2 = [
  {
    "kind": "Variable",
    "name": "input",
    "variableName": "input"
  }
],
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "status",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "decision",
  "storageKey": null
},
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "comment",
  "storageKey": null
},
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "createdBy",
  "storageKey": null
},
v8 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "coveredRules",
  "storageKey": null
},
v9 = {
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
v10 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "email",
  "storageKey": null
},
v11 = {
  "alias": null,
  "args": null,
  "concreteType": "User",
  "kind": "LinkedField",
  "name": "user",
  "plural": false,
  "selections": [
    (v3/*: any*/),
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "username",
      "storageKey": null
    },
    (v10/*: any*/)
  ],
  "storageKey": null
},
v12 = {
  "alias": null,
  "args": null,
  "concreteType": "ServiceAccount",
  "kind": "LinkedField",
  "name": "serviceAccount",
  "plural": false,
  "selections": [
    (v3/*: any*/),
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
      "name": "resourcePath",
      "storageKey": null
    }
  ],
  "storageKey": null
},
v13 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "type",
  "storageKey": null
},
v14 = {
  "alias": null,
  "args": null,
  "concreteType": "Problem",
  "kind": "LinkedField",
  "name": "problems",
  "plural": true,
  "selections": [
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
      "name": "field",
      "storageKey": null
    },
    (v13/*: any*/)
  ],
  "storageKey": null
},
v15 = [
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "trnSnapshot",
    "storageKey": null
  }
];
return {
  "fragment": {
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "RunTaskStageRunGateDecisionButtonsApproveMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "RunGateMutationPayload",
        "kind": "LinkedField",
        "name": "approveRunGate",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "RunGate",
            "kind": "LinkedField",
            "name": "runGate",
            "plural": false,
            "selections": [
              (v3/*: any*/),
              (v4/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "RunGateApproval",
                "kind": "LinkedField",
                "name": "approvals",
                "plural": true,
                "selections": [
                  (v3/*: any*/),
                  (v5/*: any*/),
                  (v6/*: any*/),
                  (v7/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": null,
                    "kind": "LinkedField",
                    "name": "createdByActor",
                    "plural": false,
                    "selections": [
                      {
                        "args": null,
                        "kind": "FragmentSpread",
                        "name": "ActorDisplayFragment_actor"
                      }
                    ],
                    "storageKey": null
                  },
                  (v8/*: any*/),
                  (v9/*: any*/),
                  (v11/*: any*/),
                  (v12/*: any*/)
                ],
                "storageKey": null
              }
            ],
            "storageKey": null
          },
          (v14/*: any*/)
        ],
        "storageKey": null
      }
    ],
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [
      (v1/*: any*/),
      (v0/*: any*/)
    ],
    "kind": "Operation",
    "name": "RunTaskStageRunGateDecisionButtonsApproveMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "RunGateMutationPayload",
        "kind": "LinkedField",
        "name": "approveRunGate",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "RunGate",
            "kind": "LinkedField",
            "name": "runGate",
            "plural": false,
            "selections": [
              (v3/*: any*/),
              {
                "alias": null,
                "args": null,
                "filters": null,
                "handle": "deleteEdge",
                "key": "",
                "kind": "ScalarHandle",
                "name": "id",
                "handleArgs": [
                  {
                    "kind": "Variable",
                    "name": "connections",
                    "variableName": "connections"
                  }
                ]
              },
              (v4/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "RunGateApproval",
                "kind": "LinkedField",
                "name": "approvals",
                "plural": true,
                "selections": [
                  (v3/*: any*/),
                  (v5/*: any*/),
                  (v6/*: any*/),
                  (v7/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": null,
                    "kind": "LinkedField",
                    "name": "createdByActor",
                    "plural": false,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "__typename",
                        "storageKey": null
                      },
                      {
                        "kind": "TypeDiscriminator",
                        "abstractKey": "__isActor"
                      },
                      (v13/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "principalId",
                        "storageKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": "User",
                            "kind": "LinkedField",
                            "name": "user",
                            "plural": false,
                            "selections": [
                              (v10/*: any*/),
                              (v3/*: any*/)
                            ],
                            "storageKey": null
                          }
                        ],
                        "type": "UserActor",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v15/*: any*/),
                        "type": "ServiceAccountActor",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v15/*: any*/),
                        "type": "VCSProviderActor",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v15/*: any*/),
                        "type": "JobActor",
                        "abstractKey": null
                      }
                    ],
                    "storageKey": null
                  },
                  (v8/*: any*/),
                  (v9/*: any*/),
                  (v11/*: any*/),
                  (v12/*: any*/)
                ],
                "storageKey": null
              }
            ],
            "storageKey": null
          },
          (v14/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "8a371ce7633173051eb13efaf4893cf1",
    "id": null,
    "metadata": {},
    "name": "RunTaskStageRunGateDecisionButtonsApproveMutation",
    "operationKind": "mutation",
    "text": "mutation RunTaskStageRunGateDecisionButtonsApproveMutation(\n  $input: ApproveRunGateInput!\n) {\n  approveRunGate(input: $input) {\n    runGate {\n      id\n      status\n      approvals {\n        id\n        decision\n        comment\n        createdBy\n        createdByActor {\n          __typename\n          ...ActorDisplayFragment_actor\n        }\n        coveredRules\n        metadata {\n          createdAt\n        }\n        user {\n          id\n          username\n          email\n        }\n        serviceAccount {\n          id\n          name\n          resourcePath\n        }\n      }\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment ActorDisplayFragment_actor on Actor {\n  __isActor: __typename\n  type\n  principalId\n  ... on UserActor {\n    user {\n      email\n      id\n    }\n  }\n  ... on ServiceAccountActor {\n    trnSnapshot\n  }\n  ... on VCSProviderActor {\n    trnSnapshot\n  }\n  ... on JobActor {\n    trnSnapshot\n  }\n}\n"
  }
};
})();

(node as any).hash = "6d2f55b050301427d3fd750285d26b88";

export default node;
