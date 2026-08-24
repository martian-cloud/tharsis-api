/**
 * @generated SignedSource<<8c848eb926f156d4f327ccea44532975>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type DeletePolicyInput = {
  clientMutationId?: string | null | undefined;
  id: string;
};
export type PolicyDetailsDeleteMutation$variables = {
  connections: ReadonlyArray<string>;
  input: DeletePolicyInput;
};
export type PolicyDetailsDeleteMutation$data = {
  readonly deletePolicy: {
    readonly policy: {
      readonly id: string;
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type PolicyDetailsDeleteMutation = {
  response: PolicyDetailsDeleteMutation$data;
  variables: PolicyDetailsDeleteMutation$variables;
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
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "type",
      "storageKey": null
    }
  ],
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "PolicyDetailsDeleteMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "PolicyMutationPayload",
        "kind": "LinkedField",
        "name": "deletePolicy",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "Policy",
            "kind": "LinkedField",
            "name": "policy",
            "plural": false,
            "selections": [
              (v3/*: any*/)
            ],
            "storageKey": null
          },
          (v4/*: any*/)
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
    "name": "PolicyDetailsDeleteMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "PolicyMutationPayload",
        "kind": "LinkedField",
        "name": "deletePolicy",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "Policy",
            "kind": "LinkedField",
            "name": "policy",
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
              }
            ],
            "storageKey": null
          },
          (v4/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "0b15d9e7a67b560f916f42dc0c25712f",
    "id": null,
    "metadata": {},
    "name": "PolicyDetailsDeleteMutation",
    "operationKind": "mutation",
    "text": "mutation PolicyDetailsDeleteMutation(\n  $input: DeletePolicyInput!\n) {\n  deletePolicy(input: $input) {\n    policy {\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "7cc8f139fcdcc3fd1597d5632cbea2ad";

export default node;
