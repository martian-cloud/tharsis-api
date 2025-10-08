/**
 * @generated SignedSource<<134ca7affc98dae8381db99300fb8b16>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type MoveManagedIdentityInput = {
  clientMutationId?: string | null | undefined;
  managedIdentityId: string;
  newParentId?: string | null | undefined;
  newParentPath?: string | null | undefined;
};
export type MoveManagedIdentityDialogMutation$variables = {
  connections: ReadonlyArray<string>;
  input: MoveManagedIdentityInput;
};
export type MoveManagedIdentityDialogMutation$data = {
  readonly moveManagedIdentity: {
    readonly managedIdentity: {
      readonly groupPath: string;
      readonly id: string;
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type MoveManagedIdentityDialogMutation = {
  response: MoveManagedIdentityDialogMutation$data;
  variables: MoveManagedIdentityDialogMutation$variables;
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
  "name": "groupPath",
  "storageKey": null
},
v5 = {
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
    "name": "MoveManagedIdentityDialogMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "MoveManagedIdentityPayload",
        "kind": "LinkedField",
        "name": "moveManagedIdentity",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "ManagedIdentity",
            "kind": "LinkedField",
            "name": "managedIdentity",
            "plural": false,
            "selections": [
              (v3/*: any*/),
              (v4/*: any*/)
            ],
            "storageKey": null
          },
          (v5/*: any*/)
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
    "name": "MoveManagedIdentityDialogMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "MoveManagedIdentityPayload",
        "kind": "LinkedField",
        "name": "moveManagedIdentity",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "ManagedIdentity",
            "kind": "LinkedField",
            "name": "managedIdentity",
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
              (v4/*: any*/)
            ],
            "storageKey": null
          },
          (v5/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "d76bf60be10708ec2ee9a7857540b7a0",
    "id": null,
    "metadata": {},
    "name": "MoveManagedIdentityDialogMutation",
    "operationKind": "mutation",
    "text": "mutation MoveManagedIdentityDialogMutation(\n  $input: MoveManagedIdentityInput!\n) {\n  moveManagedIdentity(input: $input) {\n    managedIdentity {\n      id\n      groupPath\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "66a0fc8e1de11dda2f3904648b2ebbde";

export default node;
