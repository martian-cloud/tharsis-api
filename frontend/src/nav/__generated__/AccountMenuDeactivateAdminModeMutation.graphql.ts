/**
 * @generated SignedSource<<2478658fdb662c541882c3cdaf85d14e>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type AccountMenuDeactivateAdminModeMutation$variables = Record<PropertyKey, never>;
export type AccountMenuDeactivateAdminModeMutation$data = {
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
export type AccountMenuDeactivateAdminModeMutation = {
  response: AccountMenuDeactivateAdminModeMutation$data;
  variables: AccountMenuDeactivateAdminModeMutation$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "alias": null,
    "args": null,
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
    "argumentDefinitions": [],
    "kind": "Fragment",
    "metadata": null,
    "name": "AccountMenuDeactivateAdminModeMutation",
    "selections": (v0/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [],
    "kind": "Operation",
    "name": "AccountMenuDeactivateAdminModeMutation",
    "selections": (v0/*: any*/)
  },
  "params": {
    "cacheID": "f5e1c8e02e572bbe7f7e40c09733f55e",
    "id": null,
    "metadata": {},
    "name": "AccountMenuDeactivateAdminModeMutation",
    "operationKind": "mutation",
    "text": "mutation AccountMenuDeactivateAdminModeMutation {\n  deactivateAdminMode {\n    user {\n      id\n      adminModeEnabled\n      adminModeExpiration\n    }\n    problems {\n      message\n      type\n      field\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "4bcb72a4e6714eede4eaa9dc64c49248";

export default node;
