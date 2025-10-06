/**
 * @generated SignedSource<<5614ae06254b553efa348cfd4ecd2a08>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type ResetVCSProviderOAuthTokenInput = {
  clientMutationId?: string | null | undefined;
  providerId: string;
};
export type VCSProviderDetailsResetOAuthMutation$variables = {
  input: ResetVCSProviderOAuthTokenInput;
};
export type VCSProviderDetailsResetOAuthMutation$data = {
  readonly resetVCSProviderOAuthToken: {
    readonly oAuthAuthorizationUrl: string;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type VCSProviderDetailsResetOAuthMutation = {
  response: VCSProviderDetailsResetOAuthMutation$data;
  variables: VCSProviderDetailsResetOAuthMutation$variables;
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
    "concreteType": "ResetVCSProviderOAuthTokenPayload",
    "kind": "LinkedField",
    "name": "resetVCSProviderOAuthToken",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "oAuthAuthorizationUrl",
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
    "name": "VCSProviderDetailsResetOAuthMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "VCSProviderDetailsResetOAuthMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "a76084292491ccb76014e1d7d2bca3b8",
    "id": null,
    "metadata": {},
    "name": "VCSProviderDetailsResetOAuthMutation",
    "operationKind": "mutation",
    "text": "mutation VCSProviderDetailsResetOAuthMutation(\n  $input: ResetVCSProviderOAuthTokenInput!\n) {\n  resetVCSProviderOAuthToken(input: $input) {\n    oAuthAuthorizationUrl\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "09cea609955b1fda6a56defa07c2e025";

export default node;
