/**
 * @generated SignedSource<<6de550aa3b789faa5e3161e9e5299cbd>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type CreateGPGKeyInput = {
  asciiArmor: string;
  clientMutationId?: string | null | undefined;
  groupId?: string | null | undefined;
  groupPath?: string | null | undefined;
};
export type NewGPGKeyMutation$variables = {
  connections: ReadonlyArray<string>;
  input: CreateGPGKeyInput;
};
export type NewGPGKeyMutation$data = {
  readonly createGPGKey: {
    readonly gpgKey: {
      readonly createdBy: string;
      readonly fingerprint: string;
      readonly gpgKeyId: string;
      readonly id: string;
      readonly metadata: {
        readonly createdAt: any;
      };
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type NewGPGKeyMutation = {
  response: NewGPGKeyMutation$data;
  variables: NewGPGKeyMutation$variables;
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
  "concreteType": "GPGKey",
  "kind": "LinkedField",
  "name": "gpgKey",
  "plural": false,
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "id",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "gpgKeyId",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "fingerprint",
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
    }
  ],
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
    "name": "NewGPGKeyMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "CreateGPGKeyPayload",
        "kind": "LinkedField",
        "name": "createGPGKey",
        "plural": false,
        "selections": [
          (v3/*: any*/),
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
    "name": "NewGPGKeyMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "CreateGPGKeyPayload",
        "kind": "LinkedField",
        "name": "createGPGKey",
        "plural": false,
        "selections": [
          (v3/*: any*/),
          {
            "alias": null,
            "args": null,
            "filters": null,
            "handle": "prependNode",
            "key": "",
            "kind": "LinkedHandle",
            "name": "gpgKey",
            "handleArgs": [
              {
                "kind": "Variable",
                "name": "connections",
                "variableName": "connections"
              },
              {
                "kind": "Literal",
                "name": "edgeTypeName",
                "value": "GPGKeyEdge"
              }
            ]
          },
          (v4/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "f84102d2c20ddab972d1441643d174b4",
    "id": null,
    "metadata": {},
    "name": "NewGPGKeyMutation",
    "operationKind": "mutation",
    "text": "mutation NewGPGKeyMutation(\n  $input: CreateGPGKeyInput!\n) {\n  createGPGKey(input: $input) {\n    gpgKey {\n      id\n      gpgKeyId\n      fingerprint\n      createdBy\n      metadata {\n        createdAt\n      }\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "cec8c5e70d893c296b17b3bbcc38894c";

export default node;
