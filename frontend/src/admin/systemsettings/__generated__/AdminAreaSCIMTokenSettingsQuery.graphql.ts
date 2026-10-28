/**
 * @generated SignedSource<<5ac95859eccda712d4cea6df17dd168f>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type AdminAreaSCIMTokenSettingsQuery$variables = Record<PropertyKey, never>;
export type AdminAreaSCIMTokenSettingsQuery$data = {
  readonly config: {
    readonly oauthProviders: ReadonlyArray<{
      readonly issuerUrl: string;
    }>;
  };
  readonly scimToken: {
    readonly createdByActor: {
      readonly " $fragmentSpreads": FragmentRefs<"ActorDisplayFragment_actor">;
    } | null | undefined;
    readonly metadata: {
      readonly createdAt: any;
    };
  } | null | undefined;
};
export type AdminAreaSCIMTokenSettingsQuery = {
  response: AdminAreaSCIMTokenSettingsQuery$data;
  variables: AdminAreaSCIMTokenSettingsQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = {
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
v1 = {
  "alias": null,
  "args": null,
  "concreteType": "Config",
  "kind": "LinkedField",
  "name": "config",
  "plural": false,
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "IdpConfig",
      "kind": "LinkedField",
      "name": "oauthProviders",
      "plural": true,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "issuerUrl",
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "storageKey": null
},
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v3 = [
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
    "name": "AdminAreaSCIMTokenSettingsQuery",
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "SCIMToken",
        "kind": "LinkedField",
        "name": "scimToken",
        "plural": false,
        "selections": [
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
          (v0/*: any*/)
        ],
        "storageKey": null
      },
      (v1/*: any*/)
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [],
    "kind": "Operation",
    "name": "AdminAreaSCIMTokenSettingsQuery",
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "SCIMToken",
        "kind": "LinkedField",
        "name": "scimToken",
        "plural": false,
        "selections": [
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
                "selections": (v3/*: any*/),
                "type": "ServiceAccountActor",
                "abstractKey": null
              },
              {
                "kind": "InlineFragment",
                "selections": (v3/*: any*/),
                "type": "VCSProviderActor",
                "abstractKey": null
              },
              {
                "kind": "InlineFragment",
                "selections": (v3/*: any*/),
                "type": "JobActor",
                "abstractKey": null
              }
            ],
            "storageKey": null
          },
          (v0/*: any*/),
          (v2/*: any*/)
        ],
        "storageKey": null
      },
      (v1/*: any*/)
    ]
  },
  "params": {
    "cacheID": "c4cb8d091aee1256414f0f64de57c314",
    "id": null,
    "metadata": {},
    "name": "AdminAreaSCIMTokenSettingsQuery",
    "operationKind": "query",
    "text": "query AdminAreaSCIMTokenSettingsQuery {\n  scimToken {\n    createdByActor {\n      __typename\n      ...ActorDisplayFragment_actor\n    }\n    metadata {\n      createdAt\n    }\n    id\n  }\n  config {\n    oauthProviders {\n      issuerUrl\n    }\n  }\n}\n\nfragment ActorDisplayFragment_actor on Actor {\n  __isActor: __typename\n  type\n  principalId\n  ... on UserActor {\n    user {\n      email\n      id\n    }\n  }\n  ... on ServiceAccountActor {\n    trnSnapshot\n  }\n  ... on VCSProviderActor {\n    trnSnapshot\n  }\n  ... on JobActor {\n    trnSnapshot\n  }\n}\n"
  }
};
})();

(node as any).hash = "c9e9b6b7fb020b448a626322b4222dfa";

export default node;
