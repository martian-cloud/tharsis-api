/**
 * @generated SignedSource<<4a62c3ac61e23fa876ae6f78782a9d66>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type DeleteWorkspaceVCSProviderLinkInput = {
  clientMutationId?: string | null | undefined;
  force?: boolean | null | undefined;
  id: string;
  metadata?: ResourceMetadataInput | null | undefined;
};
export type ResourceMetadataInput = {
  version: string;
};
export type VCSProviderLinkDangerZoneDeleteMutation$variables = {
  input: DeleteWorkspaceVCSProviderLinkInput;
};
export type VCSProviderLinkDangerZoneDeleteMutation$data = {
  readonly deleteWorkspaceVCSProviderLink: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly vcsProviderLink: {
      readonly workspace: {
        readonly id: string;
        readonly workspaceVcsProviderLink: {
          readonly id: string;
        } | null | undefined;
      };
    } | null | undefined;
  };
};
export type VCSProviderLinkDangerZoneDeleteMutation = {
  response: VCSProviderLinkDangerZoneDeleteMutation$data;
  variables: VCSProviderLinkDangerZoneDeleteMutation$variables;
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
  "name": "id",
  "storageKey": null
},
v3 = {
  "alias": null,
  "args": null,
  "concreteType": "Workspace",
  "kind": "LinkedField",
  "name": "workspace",
  "plural": false,
  "selections": [
    (v2/*:: as any*/),
    {
      "alias": null,
      "args": null,
      "concreteType": "WorkspaceVCSProviderLink",
      "kind": "LinkedField",
      "name": "workspaceVcsProviderLink",
      "plural": false,
      "selections": [
        (v2/*:: as any*/)
      ],
      "storageKey": null
    }
  ],
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
    "argumentDefinitions": (v0/*:: as any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "VCSProviderLinkDangerZoneDeleteMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*:: as any*/),
        "concreteType": "DeleteWorkspaceVCSProviderLinkPayload",
        "kind": "LinkedField",
        "name": "deleteWorkspaceVCSProviderLink",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "WorkspaceVCSProviderLink",
            "kind": "LinkedField",
            "name": "vcsProviderLink",
            "plural": false,
            "selections": [
              (v3/*:: as any*/)
            ],
            "storageKey": null
          },
          (v4/*:: as any*/)
        ],
        "storageKey": null
      }
    ],
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*:: as any*/),
    "kind": "Operation",
    "name": "VCSProviderLinkDangerZoneDeleteMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*:: as any*/),
        "concreteType": "DeleteWorkspaceVCSProviderLinkPayload",
        "kind": "LinkedField",
        "name": "deleteWorkspaceVCSProviderLink",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "WorkspaceVCSProviderLink",
            "kind": "LinkedField",
            "name": "vcsProviderLink",
            "plural": false,
            "selections": [
              (v3/*:: as any*/),
              (v2/*:: as any*/)
            ],
            "storageKey": null
          },
          (v4/*:: as any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "44921bf312c622d4263c1e70fbb53048",
    "id": null,
    "metadata": {},
    "name": "VCSProviderLinkDangerZoneDeleteMutation",
    "operationKind": "mutation",
    "text": "mutation VCSProviderLinkDangerZoneDeleteMutation(\n  $input: DeleteWorkspaceVCSProviderLinkInput!\n) {\n  deleteWorkspaceVCSProviderLink(input: $input) {\n    vcsProviderLink {\n      workspace {\n        id\n        workspaceVcsProviderLink {\n          id\n        }\n      }\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "89591479fb176b49489a29d2ecf6b870";

export default node;
