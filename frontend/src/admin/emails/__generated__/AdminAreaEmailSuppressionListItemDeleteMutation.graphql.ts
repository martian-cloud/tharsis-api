/**
 * @generated SignedSource<<31fe50e5b7001a3255b67178dd311e02>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type DeleteEmailSuppressionInput = {
  clientMutationId?: string | null | undefined;
  id: string;
  metadata?: ResourceMetadataInput | null | undefined;
};
export type ResourceMetadataInput = {
  version: string;
};
export type AdminAreaEmailSuppressionListItemDeleteMutation$variables = {
  connections: ReadonlyArray<string>;
  input: DeleteEmailSuppressionInput;
};
export type AdminAreaEmailSuppressionListItemDeleteMutation$data = {
  readonly deleteEmailSuppression: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly suppression: {
      readonly id: string;
    } | null | undefined;
  };
};
export type AdminAreaEmailSuppressionListItemDeleteMutation = {
  response: AdminAreaEmailSuppressionListItemDeleteMutation$data;
  variables: AdminAreaEmailSuppressionListItemDeleteMutation$variables;
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
    "name": "AdminAreaEmailSuppressionListItemDeleteMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "DeleteEmailSuppressionPayload",
        "kind": "LinkedField",
        "name": "deleteEmailSuppression",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "EmailSuppression",
            "kind": "LinkedField",
            "name": "suppression",
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
    "name": "AdminAreaEmailSuppressionListItemDeleteMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "DeleteEmailSuppressionPayload",
        "kind": "LinkedField",
        "name": "deleteEmailSuppression",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "EmailSuppression",
            "kind": "LinkedField",
            "name": "suppression",
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
    "cacheID": "3b1533d6051515dc2311cd70be1dee0c",
    "id": null,
    "metadata": {},
    "name": "AdminAreaEmailSuppressionListItemDeleteMutation",
    "operationKind": "mutation",
    "text": "mutation AdminAreaEmailSuppressionListItemDeleteMutation(\n  $input: DeleteEmailSuppressionInput!\n) {\n  deleteEmailSuppression(input: $input) {\n    suppression {\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "d7e3d1d07c2a33eada24652c1518daa5";

export default node;
