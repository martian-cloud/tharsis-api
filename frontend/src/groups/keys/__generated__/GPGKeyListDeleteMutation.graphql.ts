/**
 * @generated SignedSource<<670dc6a772b925181b34cc96374a9160>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type DeleteGPGKeyInput = {
  clientMutationId?: string | null | undefined;
  id: string;
  metadata?: ResourceMetadataInput | null | undefined;
};
export type ResourceMetadataInput = {
  version: string;
};
export type GPGKeyListDeleteMutation$variables = {
  connections: ReadonlyArray<string>;
  input: DeleteGPGKeyInput;
};
export type GPGKeyListDeleteMutation$data = {
  readonly deleteGPGKey: {
    readonly gpgKey: {
      readonly id: string;
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type GPGKeyListDeleteMutation = {
  response: GPGKeyListDeleteMutation$data;
  variables: GPGKeyListDeleteMutation$variables;
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
    "name": "GPGKeyListDeleteMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "DeleteGPGKeyPayload",
        "kind": "LinkedField",
        "name": "deleteGPGKey",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "GPGKey",
            "kind": "LinkedField",
            "name": "gpgKey",
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
    "name": "GPGKeyListDeleteMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "DeleteGPGKeyPayload",
        "kind": "LinkedField",
        "name": "deleteGPGKey",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "GPGKey",
            "kind": "LinkedField",
            "name": "gpgKey",
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
    "cacheID": "e8710efd6aa5a7ab57d6ec903cadeee1",
    "id": null,
    "metadata": {},
    "name": "GPGKeyListDeleteMutation",
    "operationKind": "mutation",
    "text": "mutation GPGKeyListDeleteMutation(\n  $input: DeleteGPGKeyInput!\n) {\n  deleteGPGKey(input: $input) {\n    gpgKey {\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "ae9c8e58a77d2546e1851285e11e53dc";

export default node;
