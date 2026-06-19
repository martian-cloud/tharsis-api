/**
 * @generated SignedSource<<745d6515c3763f7a78299b4f7136ebb8>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type ActivateAdminModeInput = {
  clientMutationId?: string | null | undefined;
  durationMinutes?: number | null | undefined;
};
export type ActivateAdminModeDialogMutation$variables = {
  input: ActivateAdminModeInput;
};
export type ActivateAdminModeDialogMutation$data = {
  readonly activateAdminMode: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly user: {
      readonly adminModeEnabled: boolean;
      readonly adminModeExpiration: any | null | undefined;
      readonly id: string;
    } | null | undefined;
  };
};
export type ActivateAdminModeDialogMutation = {
  response: ActivateAdminModeDialogMutation$data;
  variables: ActivateAdminModeDialogMutation$variables;
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
    "alias": null,
    "args": [
      {
        "kind": "Variable",
        "name": "input",
        "variableName": "input"
      }
    ],
    "concreteType": "UpdateUserAdminStatusPayload",
    "kind": "LinkedField",
    "name": "activateAdminMode",
    "plural": false,
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
            "name": "id",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "adminModeEnabled",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "adminModeExpiration",
            "storageKey": null
          }
        ],
        "storageKey": null
      },
      {
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
            "name": "type",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "field",
            "storageKey": null
          }
        ],
        "storageKey": null
      }
    ],
    "storageKey": null
  }
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "ActivateAdminModeDialogMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "ActivateAdminModeDialogMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "2f70083f46d4a68c18b430d03e721718",
    "id": null,
    "metadata": {},
    "name": "ActivateAdminModeDialogMutation",
    "operationKind": "mutation",
    "text": "mutation ActivateAdminModeDialogMutation(\n  $input: ActivateAdminModeInput!\n) {\n  activateAdminMode(input: $input) {\n    user {\n      id\n      adminModeEnabled\n      adminModeExpiration\n    }\n    problems {\n      message\n      type\n      field\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "b0ce9b8207cc68aaf47d8b05a37aed2b";

export default node;
