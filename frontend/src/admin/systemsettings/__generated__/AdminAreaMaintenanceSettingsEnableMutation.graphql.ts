/**
 * @generated SignedSource<<675589bc559692d55fb6dee4f4ac6738>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type EnableMaintenanceModeInput = {
  clientMutationId?: string | null | undefined;
};
export type AdminAreaMaintenanceSettingsEnableMutation$variables = {
  input: EnableMaintenanceModeInput;
};
export type AdminAreaMaintenanceSettingsEnableMutation$data = {
  readonly enableMaintenanceMode: {
    readonly maintenanceMode: {
      readonly createdByActor: {
        readonly " $fragmentSpreads": FragmentRefs<"ActorDisplayFragment_actor">;
      } | null | undefined;
      readonly id: string;
      readonly metadata: {
        readonly createdAt: any;
      };
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
    }>;
  };
};
export type AdminAreaMaintenanceSettingsEnableMutation = {
  response: AdminAreaMaintenanceSettingsEnableMutation$data;
  variables: AdminAreaMaintenanceSettingsEnableMutation$variables;
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
  "concreteType": "ResourceMetadata",
  "kind": "LinkedField",
  "name": "metadata",
  "plural": false,
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "createdAt",
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
    }
  ],
  "storageKey": null
},
v5 = [
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "trnSnapshot",
    "storageKey": null
  }
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "AdminAreaMaintenanceSettingsEnableMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "EnableMaintenanceModePayload",
        "kind": "LinkedField",
        "name": "enableMaintenanceMode",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "MaintenanceMode",
            "kind": "LinkedField",
            "name": "maintenanceMode",
            "plural": false,
            "selections": [
              (v2/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": null,
                "kind": "LinkedField",
                "name": "createdByActor",
                "plural": false,
                "selections": [
                  {
                    "args": null,
                    "kind": "FragmentSpread",
                    "name": "ActorDisplayFragment_actor"
                  }
                ],
                "storageKey": null
              },
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
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "AdminAreaMaintenanceSettingsEnableMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "EnableMaintenanceModePayload",
        "kind": "LinkedField",
        "name": "enableMaintenanceMode",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "MaintenanceMode",
            "kind": "LinkedField",
            "name": "maintenanceMode",
            "plural": false,
            "selections": [
              (v2/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": null,
                "kind": "LinkedField",
                "name": "createdByActor",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "__typename",
                    "storageKey": null
                  },
                  {
                    "kind": "TypeDiscriminator",
                    "abstractKey": "__isActor"
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
                    "name": "principalId",
                    "storageKey": null
                  },
                  {
                    "kind": "InlineFragment",
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
                            "name": "email",
                            "storageKey": null
                          },
                          (v2/*: any*/)
                        ],
                        "storageKey": null
                      }
                    ],
                    "type": "UserActor",
                    "abstractKey": null
                  },
                  {
                    "kind": "InlineFragment",
                    "selections": (v5/*: any*/),
                    "type": "ServiceAccountActor",
                    "abstractKey": null
                  },
                  {
                    "kind": "InlineFragment",
                    "selections": (v5/*: any*/),
                    "type": "VCSProviderActor",
                    "abstractKey": null
                  },
                  {
                    "kind": "InlineFragment",
                    "selections": (v5/*: any*/),
                    "type": "JobActor",
                    "abstractKey": null
                  }
                ],
                "storageKey": null
              },
              (v3/*: any*/)
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
    "cacheID": "6f79cd12ceb178027b00a2dde79f34c8",
    "id": null,
    "metadata": {},
    "name": "AdminAreaMaintenanceSettingsEnableMutation",
    "operationKind": "mutation",
    "text": "mutation AdminAreaMaintenanceSettingsEnableMutation(\n  $input: EnableMaintenanceModeInput!\n) {\n  enableMaintenanceMode(input: $input) {\n    maintenanceMode {\n      id\n      createdByActor {\n        __typename\n        ...ActorDisplayFragment_actor\n      }\n      metadata {\n        createdAt\n      }\n    }\n    problems {\n      message\n      field\n    }\n  }\n}\n\nfragment ActorDisplayFragment_actor on Actor {\n  __isActor: __typename\n  type\n  principalId\n  ... on UserActor {\n    user {\n      email\n      id\n    }\n  }\n  ... on ServiceAccountActor {\n    trnSnapshot\n  }\n  ... on VCSProviderActor {\n    trnSnapshot\n  }\n  ... on JobActor {\n    trnSnapshot\n  }\n}\n"
  }
};
})();

(node as any).hash = "1ef176a72bfc4632dfcbc05b4d03c979";

export default node;
