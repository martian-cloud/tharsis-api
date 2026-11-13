/**
 * @generated SignedSource<<1ad1435ae8f5e0d706e2c30e62e1c001>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type DeleteVCSProviderInput = {
  clientMutationId?: string | null | undefined;
  force?: boolean | null | undefined;
  id: string;
  metadata?: ResourceMetadataInput | null | undefined;
};
export type ResourceMetadataInput = {
  version: string;
};
export type VCSProviderDetailsDeleteMutation$variables = {
  connections: ReadonlyArray<string>;
  input: DeleteVCSProviderInput;
};
export type VCSProviderDetailsDeleteMutation$data = {
  readonly deleteVCSProvider: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly vcsProvider: {
      readonly id: string;
    } | null | undefined;
  };
};
export type VCSProviderDetailsDeleteMutation = {
  response: VCSProviderDetailsDeleteMutation$data;
  variables: VCSProviderDetailsDeleteMutation$variables;
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
      (v0/*:: as any*/),
      (v1/*:: as any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "VCSProviderDetailsDeleteMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*:: as any*/),
        "concreteType": "DeleteVCSProviderPayload",
        "kind": "LinkedField",
        "name": "deleteVCSProvider",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "VCSProvider",
            "kind": "LinkedField",
            "name": "vcsProvider",
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
    "argumentDefinitions": [
      (v1/*:: as any*/),
      (v0/*:: as any*/)
    ],
    "kind": "Operation",
    "name": "VCSProviderDetailsDeleteMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*:: as any*/),
        "concreteType": "DeleteVCSProviderPayload",
        "kind": "LinkedField",
        "name": "deleteVCSProvider",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "VCSProvider",
            "kind": "LinkedField",
            "name": "vcsProvider",
            "plural": false,
            "selections": [
              (v3/*:: as any*/),
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
          (v4/*:: as any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "fa45f248170b556d4b8e4a520bbaa0fb",
    "id": null,
    "metadata": {},
    "name": "VCSProviderDetailsDeleteMutation",
    "operationKind": "mutation",
    "text": "mutation VCSProviderDetailsDeleteMutation(\n  $input: DeleteVCSProviderInput!\n) {\n  deleteVCSProvider(input: $input) {\n    vcsProvider {\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "fdb3519bce52b24b489291e5e6859ea9";

export default node;
