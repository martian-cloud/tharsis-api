/**
 * @generated SignedSource<<e8073cd7009a2e0798ad83e38eaa008f>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type AnnouncementType = "ERROR" | "INFO" | "SUCCESS" | "WARNING" | "%future added value";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type CreateAnnouncementInput = {
  clientMutationId?: string | null | undefined;
  dismissible: boolean;
  endTime?: any | null | undefined;
  message: string;
  startTime?: any | null | undefined;
  type: AnnouncementType;
};
export type AdminAreaNewAnnouncementMutation$variables = {
  input: CreateAnnouncementInput;
};
export type AdminAreaNewAnnouncementMutation$data = {
  readonly createAnnouncement: {
    readonly announcement: {
      readonly dismissible: boolean;
      readonly endTime: any | null | undefined;
      readonly id: string;
      readonly message: string;
      readonly startTime: any;
      readonly type: AnnouncementType;
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type AdminAreaNewAnnouncementMutation = {
  response: AdminAreaNewAnnouncementMutation$data;
  variables: AdminAreaNewAnnouncementMutation$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "input"
  }
],
v1 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "message",
  "storageKey": null
},
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "type",
  "storageKey": null
},
v3 = [
  {
    "alias": null,
    "args": [
      {
        "kind": "Variable",
        "name": "input",
        "variableName": "input"
      }
    ],
    "concreteType": "CreateAnnouncementPayload",
    "kind": "LinkedField",
    "name": "createAnnouncement",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "Announcement",
        "kind": "LinkedField",
        "name": "announcement",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "id",
            "storageKey": null
          },
          (v1/*: any*/),
          (v2/*: any*/),
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "dismissible",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "startTime",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "endTime",
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
          (v1/*: any*/),
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "field",
            "storageKey": null
          },
          (v2/*: any*/)
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
    "name": "AdminAreaNewAnnouncementMutation",
    "selections": (v3/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "AdminAreaNewAnnouncementMutation",
    "selections": (v3/*: any*/)
  },
  "params": {
    "cacheID": "20a67634bf764a16accb772929ab062b",
    "id": null,
    "metadata": {},
    "name": "AdminAreaNewAnnouncementMutation",
    "operationKind": "mutation",
    "text": "mutation AdminAreaNewAnnouncementMutation(\n  $input: CreateAnnouncementInput!\n) {\n  createAnnouncement(input: $input) {\n    announcement {\n      id\n      message\n      type\n      dismissible\n      startTime\n      endTime\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "cfefa0ad485458c677aea7d48f7c81e2";

export default node;
