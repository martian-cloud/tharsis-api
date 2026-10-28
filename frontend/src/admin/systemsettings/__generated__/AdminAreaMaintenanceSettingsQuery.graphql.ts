/**
 * @generated SignedSource<<e54abf2bd67efde849c4377f72cddabe>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type AdminAreaMaintenanceSettingsQuery$variables = Record<PropertyKey, never>;
export type AdminAreaMaintenanceSettingsQuery$data = {
  readonly maintenanceMode: {
    readonly createdByActor: {
      readonly " $fragmentSpreads": FragmentRefs<"ActorDisplayFragment_actor">;
    } | null | undefined;
    readonly id: string;
    readonly metadata: {
      readonly createdAt: any;
    };
  } | null | undefined;
};
export type AdminAreaMaintenanceSettingsQuery = {
  response: AdminAreaMaintenanceSettingsQuery$data;
  variables: AdminAreaMaintenanceSettingsQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v1 = {
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
v2 = [
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
    "argumentDefinitions": [],
    "kind": "Fragment",
    "metadata": null,
    "name": "AdminAreaMaintenanceSettingsQuery",
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "MaintenanceMode",
        "kind": "LinkedField",
        "name": "maintenanceMode",
        "plural": false,
        "selections": [
          (v0/*: any*/),
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
          (v1/*: any*/)
        ],
        "storageKey": null
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [],
    "kind": "Operation",
    "name": "AdminAreaMaintenanceSettingsQuery",
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "MaintenanceMode",
        "kind": "LinkedField",
        "name": "maintenanceMode",
        "plural": false,
        "selections": [
          (v0/*: any*/),
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
                      (v0/*: any*/)
                    ],
                    "storageKey": null
                  }
                ],
                "type": "UserActor",
                "abstractKey": null
              },
              {
                "kind": "InlineFragment",
                "selections": (v2/*: any*/),
                "type": "ServiceAccountActor",
                "abstractKey": null
              },
              {
                "kind": "InlineFragment",
                "selections": (v2/*: any*/),
                "type": "VCSProviderActor",
                "abstractKey": null
              },
              {
                "kind": "InlineFragment",
                "selections": (v2/*: any*/),
                "type": "JobActor",
                "abstractKey": null
              }
            ],
            "storageKey": null
          },
          (v1/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "752e8634ec9c53a2a83e2990dbbd417f",
    "id": null,
    "metadata": {},
    "name": "AdminAreaMaintenanceSettingsQuery",
    "operationKind": "query",
    "text": "query AdminAreaMaintenanceSettingsQuery {\n  maintenanceMode {\n    id\n    createdByActor {\n      __typename\n      ...ActorDisplayFragment_actor\n    }\n    metadata {\n      createdAt\n    }\n  }\n}\n\nfragment ActorDisplayFragment_actor on Actor {\n  __isActor: __typename\n  type\n  principalId\n  ... on UserActor {\n    user {\n      email\n      id\n    }\n  }\n  ... on ServiceAccountActor {\n    trnSnapshot\n  }\n  ... on VCSProviderActor {\n    trnSnapshot\n  }\n  ... on JobActor {\n    trnSnapshot\n  }\n}\n"
  }
};
})();

(node as any).hash = "69374e6a58505cb819ff8b3116c02088";

export default node;
