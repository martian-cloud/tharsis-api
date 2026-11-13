/**
 * @generated SignedSource<<5e6ce2ae5dce0ab9b69337ce9e5dea5a>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type RunGateStatus = "APPROVED" | "CANCELED" | "OVERRIDDEN" | "PENDING" | "%future added value";
export type OverrideRunGateInput = {
  clientMutationId?: string | null | undefined;
  comment?: string | null | undefined;
  gateId: string;
};
export type RunTaskStageOverrideRunGateDialogMutation$variables = {
  input: OverrideRunGateInput;
};
export type RunTaskStageOverrideRunGateDialogMutation$data = {
  readonly overrideRunGate: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly runGate: {
      readonly id: string;
      readonly metadata: {
        readonly updatedAt: any;
      };
      readonly overriddenBy: string | null | undefined;
      readonly overriddenByActor: {
        readonly " $fragmentSpreads": FragmentRefs<"ActorDisplayFragment_actor">;
      } | null | undefined;
      readonly overrideComment: string;
      readonly status: RunGateStatus;
    } | null | undefined;
  };
};
export type RunTaskStageOverrideRunGateDialogMutation = {
  response: RunTaskStageOverrideRunGateDialogMutation$data;
  variables: RunTaskStageOverrideRunGateDialogMutation$variables;
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
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "overriddenBy",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "overrideComment",
  "storageKey": null
},
v6 = {
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
      "name": "updatedAt",
      "storageKey": null
    }
  ],
  "storageKey": null
},
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "type",
  "storageKey": null
},
v8 = {
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
    (v7/*:: as any*/)
  ],
  "storageKey": null
},
v9 = [
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
    "argumentDefinitions": (v0/*:: as any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "RunTaskStageOverrideRunGateDialogMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*:: as any*/),
        "concreteType": "RunGateMutationPayload",
        "kind": "LinkedField",
        "name": "overrideRunGate",
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
              (v2/*:: as any*/),
              (v3/*:: as any*/),
              (v4/*:: as any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": null,
                "kind": "LinkedField",
                "name": "overriddenByActor",
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
              (v5/*:: as any*/),
              (v6/*:: as any*/)
            ],
            "storageKey": null
          },
          (v8/*:: as any*/)
        ],
        "storageKey": null
      }
    ],
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*:: as any*/),
    "kind": "Operation",
    "name": "RunTaskStageOverrideRunGateDialogMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*:: as any*/),
        "concreteType": "RunGateMutationPayload",
        "kind": "LinkedField",
        "name": "overrideRunGate",
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
              (v2/*:: as any*/),
              (v3/*:: as any*/),
              (v4/*:: as any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": null,
                "kind": "LinkedField",
                "name": "overriddenByActor",
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
                  (v7/*:: as any*/),
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
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "email",
                            "storageKey": null
                          },
                          (v2/*:: as any*/)
                        ],
                        "storageKey": null
                      }
                    ],
                    "type": "UserActor",
                    "abstractKey": null
                  },
                  {
                    "kind": "InlineFragment",
                    "selections": (v9/*:: as any*/),
                    "type": "ServiceAccountActor",
                    "abstractKey": null
                  },
                  {
                    "kind": "InlineFragment",
                    "selections": (v9/*:: as any*/),
                    "type": "VCSProviderActor",
                    "abstractKey": null
                  },
                  {
                    "kind": "InlineFragment",
                    "selections": (v9/*:: as any*/),
                    "type": "JobActor",
                    "abstractKey": null
                  }
                ],
                "storageKey": null
              },
              (v5/*:: as any*/),
              (v6/*:: as any*/)
            ],
            "storageKey": null
          },
          (v8/*:: as any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "c3672d0f9db1a41d70ffba2ecdb3bf85",
    "id": null,
    "metadata": {},
    "name": "RunTaskStageOverrideRunGateDialogMutation",
    "operationKind": "mutation",
    "text": "mutation RunTaskStageOverrideRunGateDialogMutation(\n  $input: OverrideRunGateInput!\n) {\n  overrideRunGate(input: $input) {\n    runGate {\n      id\n      status\n      overriddenBy\n      overriddenByActor {\n        __typename\n        ...ActorDisplayFragment_actor\n      }\n      overrideComment\n      metadata {\n        updatedAt\n      }\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment ActorDisplayFragment_actor on Actor {\n  __isActor: __typename\n  type\n  principalId\n  ... on UserActor {\n    user {\n      email\n      id\n    }\n  }\n  ... on ServiceAccountActor {\n    trnSnapshot\n  }\n  ... on VCSProviderActor {\n    trnSnapshot\n  }\n  ... on JobActor {\n    trnSnapshot\n  }\n}\n"
  }
};
})();

(node as any).hash = "1558a34f029ca22ba008e9712624ecf5";

export default node;
