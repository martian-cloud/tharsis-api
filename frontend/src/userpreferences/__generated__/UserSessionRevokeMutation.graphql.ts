/**
 * @generated SignedSource<<e0ceb381991916a4f07a96d5e89a2fe6>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type RevokeUserSessionInput = {
  userSessionId: string;
};
export type UserSessionRevokeMutation$variables = {
  input: RevokeUserSessionInput;
};
export type UserSessionRevokeMutation$data = {
  readonly revokeUserSession: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
    }>;
  };
};
export type UserSessionRevokeMutation = {
  response: UserSessionRevokeMutation$data;
  variables: UserSessionRevokeMutation$variables;
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
    "concreteType": "RevokeUserSessionPayload",
    "kind": "LinkedField",
    "name": "revokeUserSession",
    "plural": false,
    "selections": [
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
    "name": "UserSessionRevokeMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "UserSessionRevokeMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "5e9c3f656205c01b26d79817f74c97a9",
    "id": null,
    "metadata": {},
    "name": "UserSessionRevokeMutation",
    "operationKind": "mutation",
    "text": "mutation UserSessionRevokeMutation(\n  $input: RevokeUserSessionInput!\n) {\n  revokeUserSession(input: $input) {\n    problems {\n      message\n      field\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "90208869dca85843dc0bde4f1eaea8cc";

export default node;
