/**
 * @generated SignedSource<<c05a1d767ecb3868638422e9a8d962d2>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type MarkEmailRecipientClickedInput = {
  clientMutationId?: string | null | undefined;
  recipientId: string;
};
export type EmailRecipientClickTrackerMutation$variables = {
  input: MarkEmailRecipientClickedInput;
};
export type EmailRecipientClickTrackerMutation$data = {
  readonly markEmailRecipientClicked: {
    readonly emailRecipient: {
      readonly id: string;
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type EmailRecipientClickTrackerMutation = {
  response: EmailRecipientClickTrackerMutation$data;
  variables: EmailRecipientClickTrackerMutation$variables;
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
    "concreteType": "MarkEmailRecipientClickedPayload",
    "kind": "LinkedField",
    "name": "markEmailRecipientClicked",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "EmailRecipient",
        "kind": "LinkedField",
        "name": "emailRecipient",
        "plural": false,
        "selections": [
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
    "name": "EmailRecipientClickTrackerMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "EmailRecipientClickTrackerMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "309fbdbfb90d11d19d318bbc773f5d0d",
    "id": null,
    "metadata": {},
    "name": "EmailRecipientClickTrackerMutation",
    "operationKind": "mutation",
    "text": "mutation EmailRecipientClickTrackerMutation(\n  $input: MarkEmailRecipientClickedInput!\n) {\n  markEmailRecipientClicked(input: $input) {\n    emailRecipient {\n      id\n    }\n    problems {\n      message\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "879bec69c8f8a3a0cd42de1c421719c0";

export default node;
