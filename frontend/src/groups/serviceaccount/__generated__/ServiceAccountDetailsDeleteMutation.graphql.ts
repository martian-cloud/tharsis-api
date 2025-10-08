/**
 * @generated SignedSource<<17904345f40b252c8516e46016e4f786>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type DeleteServiceAccountInput = {
  clientMutationId?: string | null | undefined;
  id: string;
  metadata?: ResourceMetadataInput | null | undefined;
};
export type ResourceMetadataInput = {
  version: string;
};
export type ServiceAccountDetailsDeleteMutation$variables = {
  connections: ReadonlyArray<string>;
  input: DeleteServiceAccountInput;
};
export type ServiceAccountDetailsDeleteMutation$data = {
  readonly deleteServiceAccount: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly serviceAccount: {
      readonly id: string;
    } | null | undefined;
  };
};
export type ServiceAccountDetailsDeleteMutation = {
  response: ServiceAccountDetailsDeleteMutation$data;
  variables: ServiceAccountDetailsDeleteMutation$variables;
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
    "name": "ServiceAccountDetailsDeleteMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "DeleteServiceAccountPayload",
        "kind": "LinkedField",
        "name": "deleteServiceAccount",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "ServiceAccount",
            "kind": "LinkedField",
            "name": "serviceAccount",
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
    "name": "ServiceAccountDetailsDeleteMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "DeleteServiceAccountPayload",
        "kind": "LinkedField",
        "name": "deleteServiceAccount",
        "plural": false,
        "selections": [
          {
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
    "cacheID": "acad4e822ca93427141e6b724b0a5cab",
    "id": null,
    "metadata": {},
    "name": "ServiceAccountDetailsDeleteMutation",
    "operationKind": "mutation",
    "text": "mutation ServiceAccountDetailsDeleteMutation(\n  $input: DeleteServiceAccountInput!\n) {\n  deleteServiceAccount(input: $input) {\n    serviceAccount {\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "ef6685eaddc44fcbdc6654f7335bb764";

export default node;
