/**
 * @generated SignedSource<<ff2fa77d3c1abe05b62f900e1747bdfd>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type AnnouncementType = "ERROR" | "INFO" | "SUCCESS" | "WARNING" | "%future added value";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type UpdateAnnouncementInput = {
  clientMutationId?: string | null | undefined;
  dismissible?: boolean | null | undefined;
  endTime?: any | null | undefined;
  id: string;
  message?: string | null | undefined;
  metadata?: ResourceMetadataInput | null | undefined;
  startTime?: any | null | undefined;
  type?: AnnouncementType | null | undefined;
};
export type ResourceMetadataInput = {
  version: string;
};
export type EditAdminAreaAnnouncementMutation$variables = {
  input: UpdateAnnouncementInput;
};
export type EditAdminAreaAnnouncementMutation$data = {
  readonly updateAnnouncement: {
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
export type EditAdminAreaAnnouncementMutation = {
  response: EditAdminAreaAnnouncementMutation$data;
  variables: EditAdminAreaAnnouncementMutation$variables;
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
    "concreteType": "UpdateAnnouncementPayload",
    "kind": "LinkedField",
    "name": "updateAnnouncement",
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
    "name": "EditAdminAreaAnnouncementMutation",
    "selections": (v3/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "EditAdminAreaAnnouncementMutation",
    "selections": (v3/*: any*/)
  },
  "params": {
    "cacheID": "dbc2d3cca4c67a0cabdf3a2fafff8fee",
    "id": null,
    "metadata": {},
    "name": "EditAdminAreaAnnouncementMutation",
    "operationKind": "mutation",
    "text": "mutation EditAdminAreaAnnouncementMutation(\n  $input: UpdateAnnouncementInput!\n) {\n  updateAnnouncement(input: $input) {\n    announcement {\n      id\n      message\n      type\n      dismissible\n      startTime\n      endTime\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "fb911d7f5a1e84f7d39eba40f50df7ae";

export default node;
