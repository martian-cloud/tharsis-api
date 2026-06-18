/**
 * @generated SignedSource<<b53d72736f2a21a568bcc35c89b186df>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type DeactivateAdminModeInput = {
  clientMutationId?: string | null | undefined;
};
export type DeactivateAdminModeListItemMutation$variables = {
  input: DeactivateAdminModeInput;
};
export type DeactivateAdminModeListItemMutation$data = {
  readonly deactivateAdminMode: {
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
export type DeactivateAdminModeListItemMutation = {
  response: DeactivateAdminModeListItemMutation$data;
  variables: DeactivateAdminModeListItemMutation$variables;
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
    "name": "deactivateAdminMode",
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
    "name": "DeactivateAdminModeListItemMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "DeactivateAdminModeListItemMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "0450edd7248d3de4ebc0b6625ae86488",
    "id": null,
    "metadata": {},
    "name": "DeactivateAdminModeListItemMutation",
    "operationKind": "mutation",
    "text": "mutation DeactivateAdminModeListItemMutation(\n  $input: DeactivateAdminModeInput!\n) {\n  deactivateAdminMode(input: $input) {\n    user {\n      id\n      adminModeEnabled\n      adminModeExpiration\n    }\n    problems {\n      message\n      type\n      field\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "c87c49bcbc544cbee486b3d6901c8a84";

export default node;
