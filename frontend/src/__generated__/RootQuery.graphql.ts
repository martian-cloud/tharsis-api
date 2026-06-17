/**
 * @generated SignedSource<<06b455ab70183b96ba6da7c65cfbf07e>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type RootQuery$variables = Record<PropertyKey, never>;
export type RootQuery$data = {
  readonly config: {
    readonly aiEnabled: boolean;
    readonly serviceAccountClientSecretMaxExpirationDays: number;
    readonly serviceDiscoveryHost: string;
    readonly tharsisSupportUrl: string;
  };
  readonly me: {
    readonly admin?: boolean;
    readonly adminModeEnabled?: boolean;
    readonly adminModeExpiration?: any | null | undefined;
    readonly email?: string;
    readonly id?: string;
    readonly username?: string;
  } | null | undefined;
  readonly " $fragmentSpreads": FragmentRefs<"AppHeaderFragment">;
};
export type RootQuery = {
  response: RootQuery$data;
  variables: RootQuery$variables;
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
  "kind": "InlineFragment",
  "selections": [
    (v0/*: any*/),
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "username",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "email",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "admin",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "adminModeEnabled",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "adminModeExpiration",
      "storageKey": null
    }
  ],
  "type": "User",
  "abstractKey": null
},
v2 = {
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
      "kind": "ScalarField",
      "name": "tharsisSupportUrl",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "serviceDiscoveryHost",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "serviceAccountClientSecretMaxExpirationDays",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "aiEnabled",
      "storageKey": null
    }
  ],
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": [],
    "kind": "Fragment",
    "metadata": null,
    "name": "RootQuery",
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": null,
        "kind": "LinkedField",
        "name": "me",
        "plural": false,
        "selections": [
          (v1/*: any*/)
        ],
        "storageKey": null
      },
      (v2/*: any*/),
      {
        "args": null,
        "kind": "FragmentSpread",
        "name": "AppHeaderFragment"
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [],
    "kind": "Operation",
    "name": "RootQuery",
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": null,
        "kind": "LinkedField",
        "name": "me",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "__typename",
            "storageKey": null
          },
          (v1/*: any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              (v0/*: any*/)
            ],
            "type": "Node",
            "abstractKey": "__isNode"
          }
        ],
        "storageKey": null
      },
      (v2/*: any*/),
      {
        "alias": null,
        "args": null,
        "concreteType": "Version",
        "kind": "LinkedField",
        "name": "version",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "version",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "dbMigrationVersion",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "dbMigrationDirty",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "buildTimestamp",
            "storageKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "2ce740aa2c2534e6fbd719f578b3ef83",
    "id": null,
    "metadata": {},
    "name": "RootQuery",
    "operationKind": "query",
    "text": "query RootQuery {\n  me {\n    __typename\n    ... on User {\n      id\n      username\n      email\n      admin\n      adminModeEnabled\n      adminModeExpiration\n    }\n    ... on Node {\n      __isNode: __typename\n      id\n    }\n  }\n  config {\n    tharsisSupportUrl\n    serviceDiscoveryHost\n    serviceAccountClientSecretMaxExpirationDays\n    aiEnabled\n  }\n  ...AppHeaderFragment\n}\n\nfragment AccountMenuFragment on Query {\n  version {\n    version\n    dbMigrationVersion\n    dbMigrationDirty\n    buildTimestamp\n  }\n}\n\nfragment AppHeaderFragment on Query {\n  ...AccountMenuFragment\n}\n"
  }
};
})();

(node as any).hash = "deffe708976d37ac1baa53823a45774f";

export default node;
