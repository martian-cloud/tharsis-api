/**
 * @generated SignedSource<<17cf2fa321822fd8245ea6f42a2344dc>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type CreateSCIMTokenInput = {
  clientMutationId?: string | null | undefined;
  idpIssuerURL: string;
};
export type AdminAreaSCIMTokenSettingsMutation$variables = {
  input: CreateSCIMTokenInput;
};
export type AdminAreaSCIMTokenSettingsMutation$data = {
  readonly createSCIMToken: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly scimToken: {
      readonly createdBy: string;
      readonly metadata: {
        readonly createdAt: any;
      };
    } | null | undefined;
    readonly tokenText: string | null | undefined;
  };
};
export type AdminAreaSCIMTokenSettingsMutation = {
  response: AdminAreaSCIMTokenSettingsMutation$data;
  variables: AdminAreaSCIMTokenSettingsMutation$variables;
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
  "name": "tokenText",
  "storageKey": null
},
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "createdBy",
  "storageKey": null
},
v4 = {
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
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "AdminAreaSCIMTokenSettingsMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "CreateSCIMTokenPayload",
        "kind": "LinkedField",
        "name": "createSCIMToken",
        "plural": false,
        "selections": [
          (v2/*: any*/),
          {
            "alias": null,
            "args": null,
            "concreteType": "SCIMToken",
            "kind": "LinkedField",
            "name": "scimToken",
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
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "AdminAreaSCIMTokenSettingsMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "CreateSCIMTokenPayload",
        "kind": "LinkedField",
        "name": "createSCIMToken",
        "plural": false,
        "selections": [
          (v2/*: any*/),
          {
            "alias": null,
            "args": null,
            "concreteType": "SCIMToken",
            "kind": "LinkedField",
            "name": "scimToken",
            "plural": false,
            "selections": [
              (v3/*: any*/),
              (v4/*: any*/),
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "id",
                "storageKey": null
              }
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
    "cacheID": "90087eefa44938905418de22d29e1d2f",
    "id": null,
    "metadata": {},
    "name": "AdminAreaSCIMTokenSettingsMutation",
    "operationKind": "mutation",
    "text": "mutation AdminAreaSCIMTokenSettingsMutation(\n  $input: CreateSCIMTokenInput!\n) {\n  createSCIMToken(input: $input) {\n    tokenText\n    scimToken {\n      createdBy\n      metadata {\n        createdAt\n      }\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "159e71115a71a4e830e03eb895c8bb51";

export default node;
