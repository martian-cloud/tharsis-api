/**
 * @generated SignedSource<<997fa2910d3473d8f069d424f73e8b93>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type DeleteAnnouncementInput = {
  clientMutationId?: string | null | undefined;
  id: string;
  metadata?: ResourceMetadataInput | null | undefined;
};
export type ResourceMetadataInput = {
  version: string;
};
export type AdminAreaAnnouncementListDeleteMutation$variables = {
  connections: ReadonlyArray<string>;
  input: DeleteAnnouncementInput;
};
export type AdminAreaAnnouncementListDeleteMutation$data = {
  readonly deleteAnnouncement: {
    readonly announcement: {
      readonly id: string;
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type AdminAreaAnnouncementListDeleteMutation = {
  response: AdminAreaAnnouncementListDeleteMutation$data;
  variables: AdminAreaAnnouncementListDeleteMutation$variables;
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
    "name": "AdminAreaAnnouncementListDeleteMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "DeleteAnnouncementPayload",
        "kind": "LinkedField",
        "name": "deleteAnnouncement",
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
    "name": "AdminAreaAnnouncementListDeleteMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "DeleteAnnouncementPayload",
        "kind": "LinkedField",
        "name": "deleteAnnouncement",
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
    "cacheID": "b525abebd0df1a6b6a2a1b0f2e780acf",
    "id": null,
    "metadata": {},
    "name": "AdminAreaAnnouncementListDeleteMutation",
    "operationKind": "mutation",
    "text": "mutation AdminAreaAnnouncementListDeleteMutation(\n  $input: DeleteAnnouncementInput!\n) {\n  deleteAnnouncement(input: $input) {\n    announcement {\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "724c01bb74169d146049160aa9c9d8a1";

export default node;
