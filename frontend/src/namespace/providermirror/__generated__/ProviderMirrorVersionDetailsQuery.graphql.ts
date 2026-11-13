/**
 * @generated SignedSource<<6497a338cd66d176512ec74d9d1722ca>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ProviderMirrorVersionDetailsQuery$variables = {
  id: string;
};
export type ProviderMirrorVersionDetailsQuery$data = {
  readonly node: {
    readonly createdByActor?: {
      readonly " $fragmentSpreads": FragmentRefs<"ActorDisplayFragment_actor">;
    } | null | undefined;
    readonly groupPath?: string;
    readonly id?: string;
    readonly metadata?: {
      readonly createdAt: any;
      readonly trn: string;
    };
    readonly platformMirrors?: ReadonlyArray<{
      readonly arch: string;
      readonly id: string;
      readonly os: string;
    }>;
    readonly providerAddress?: string;
    readonly version?: string;
  } | null | undefined;
};
export type ProviderMirrorVersionDetailsQuery = {
  response: ProviderMirrorVersionDetailsQuery$data;
  variables: ProviderMirrorVersionDetailsQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "id"
  }
],
v1 = [
  {
    "kind": "Variable",
    "name": "id",
    "variableName": "id"
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
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "trn",
      "storageKey": null
    }
  ],
  "storageKey": null
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "version",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "providerAddress",
  "storageKey": null
},
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "groupPath",
  "storageKey": null
},
v7 = {
  "alias": null,
  "args": null,
  "concreteType": "TerraformProviderPlatformMirror",
  "kind": "LinkedField",
  "name": "platformMirrors",
  "plural": true,
  "selections": [
    (v2/*:: as any*/),
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "os",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "arch",
      "storageKey": null
    }
  ],
  "storageKey": null
},
v8 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "__typename",
  "storageKey": null
},
v9 = [
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
    "argumentDefinitions": (v0/*:: as any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "ProviderMirrorVersionDetailsQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*:: as any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          {
            "kind": "InlineFragment",
            "selections": [
              (v2/*:: as any*/),
              (v3/*:: as any*/),
              (v4/*:: as any*/),
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
              (v5/*:: as any*/),
              (v6/*:: as any*/),
              (v7/*:: as any*/)
            ],
            "type": "TerraformProviderVersionMirror",
            "abstractKey": null
          }
        ],
        "storageKey": null
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*:: as any*/),
    "kind": "Operation",
    "name": "ProviderMirrorVersionDetailsQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*:: as any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          (v8/*:: as any*/),
          (v2/*:: as any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              (v3/*:: as any*/),
              (v4/*:: as any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": null,
                "kind": "LinkedField",
                "name": "createdByActor",
                "plural": false,
                "selections": [
                  (v8/*:: as any*/),
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
                          (v2/*:: as any*/)
                        ],
                        "storageKey": null
                      }
                    ],
                    "type": "UserActor",
                    "abstractKey": null
                  },
                  {
                    "kind": "InlineFragment",
                    "selections": (v9/*:: as any*/),
                    "type": "ServiceAccountActor",
                    "abstractKey": null
                  },
                  {
                    "kind": "InlineFragment",
                    "selections": (v9/*:: as any*/),
                    "type": "VCSProviderActor",
                    "abstractKey": null
                  },
                  {
                    "kind": "InlineFragment",
                    "selections": (v9/*:: as any*/),
                    "type": "JobActor",
                    "abstractKey": null
                  }
                ],
                "storageKey": null
              },
              (v5/*:: as any*/),
              (v6/*:: as any*/),
              (v7/*:: as any*/)
            ],
            "type": "TerraformProviderVersionMirror",
            "abstractKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "622d1adb9f8e0bd3f96b22b4412b21ec",
    "id": null,
    "metadata": {},
    "name": "ProviderMirrorVersionDetailsQuery",
    "operationKind": "query",
    "text": "query ProviderMirrorVersionDetailsQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ... on TerraformProviderVersionMirror {\n      id\n      metadata {\n        createdAt\n        trn\n      }\n      version\n      createdByActor {\n        __typename\n        ...ActorDisplayFragment_actor\n      }\n      providerAddress\n      groupPath\n      platformMirrors {\n        id\n        os\n        arch\n      }\n    }\n    id\n  }\n}\n\nfragment ActorDisplayFragment_actor on Actor {\n  __isActor: __typename\n  type\n  principalId\n  ... on UserActor {\n    user {\n      email\n      id\n    }\n  }\n  ... on ServiceAccountActor {\n    trnSnapshot\n  }\n  ... on VCSProviderActor {\n    trnSnapshot\n  }\n  ... on JobActor {\n    trnSnapshot\n  }\n}\n"
  }
};
})();

(node as any).hash = "75d014f7a94daee1205f49e69219c87d";

export default node;
