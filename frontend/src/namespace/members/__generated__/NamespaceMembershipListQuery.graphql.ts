/**
 * @generated SignedSource<<c8ec660fc3c32d0469706a1ca6e0e785>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type NamespaceMembershipListQuery$variables = {
  id: string;
};
export type NamespaceMembershipListQuery$data = {
  readonly node: {
    readonly memberships?: ReadonlyArray<{
      readonly id: string;
      readonly member: {
        readonly __typename: "ServiceAccount";
        readonly name: string;
        readonly resourcePath: string;
      } | {
        readonly __typename: "Team";
        readonly name: string;
      } | {
        readonly __typename: "User";
        readonly email: string;
        readonly username: string;
      } | {
        // This will never be '%other', but we need some
        // value in case none of the concrete values match.
        readonly __typename: "%other";
      } | null | undefined;
      readonly " $fragmentSpreads": FragmentRefs<"NamespaceMembershipListItemFragment_membership">;
    }>;
  } | null | undefined;
};
export type NamespaceMembershipListQuery = {
  response: NamespaceMembershipListQuery$data;
  variables: NamespaceMembershipListQuery$variables;
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
  "kind": "ScalarField",
  "name": "__typename",
  "storageKey": null
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "username",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "email",
  "storageKey": null
},
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "name",
  "storageKey": null
},
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "resourcePath",
  "storageKey": null
},
v8 = [
  (v6/*: any*/),
  (v2/*: any*/)
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "NamespaceMembershipListQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          {
            "kind": "InlineFragment",
            "selections": [
              {
                "alias": null,
                "args": null,
                "concreteType": "NamespaceMembership",
                "kind": "LinkedField",
                "name": "memberships",
                "plural": true,
                "selections": [
                  (v2/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": null,
                    "kind": "LinkedField",
                    "name": "member",
                    "plural": false,
                    "selections": [
                      (v3/*: any*/),
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          (v4/*: any*/),
                          (v5/*: any*/)
                        ],
                        "type": "User",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          (v6/*: any*/)
                        ],
                        "type": "Team",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          (v7/*: any*/),
                          (v6/*: any*/)
                        ],
                        "type": "ServiceAccount",
                        "abstractKey": null
                      }
                    ],
                    "storageKey": null
                  },
                  {
                    "args": null,
                    "kind": "FragmentSpread",
                    "name": "NamespaceMembershipListItemFragment_membership"
                  }
                ],
                "storageKey": null
              }
            ],
            "type": "Namespace",
            "abstractKey": "__isNamespace"
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
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "NamespaceMembershipListQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          (v3/*: any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              {
                "alias": null,
                "args": null,
                "concreteType": "NamespaceMembership",
                "kind": "LinkedField",
                "name": "memberships",
                "plural": true,
                "selections": [
                  (v2/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": null,
                    "kind": "LinkedField",
                    "name": "member",
                    "plural": false,
                    "selections": [
                      (v3/*: any*/),
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          (v4/*: any*/),
                          (v5/*: any*/),
                          (v2/*: any*/)
                        ],
                        "type": "User",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v8/*: any*/),
                        "type": "Team",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          (v7/*: any*/),
                          (v6/*: any*/),
                          (v2/*: any*/)
                        ],
                        "type": "ServiceAccount",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          (v2/*: any*/)
                        ],
                        "type": "Node",
                        "abstractKey": "__isNode"
                      }
                    ],
                    "storageKey": null
                  },
                  {
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
                        "name": "updatedAt",
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
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "Role",
                    "kind": "LinkedField",
                    "name": "role",
                    "plural": false,
                    "selections": (v8/*: any*/),
                    "storageKey": null
                  },
                  (v7/*: any*/)
                ],
                "storageKey": null
              }
            ],
            "type": "Namespace",
            "abstractKey": "__isNamespace"
          },
          (v2/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "67272722d540837c616e5556673e9682",
    "id": null,
    "metadata": {},
    "name": "NamespaceMembershipListQuery",
    "operationKind": "query",
    "text": "query NamespaceMembershipListQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ... on Namespace {\n      __isNamespace: __typename\n      memberships {\n        id\n        member {\n          __typename\n          ... on User {\n            username\n            email\n          }\n          ... on Team {\n            name\n          }\n          ... on ServiceAccount {\n            resourcePath\n            name\n          }\n          ... on Node {\n            __isNode: __typename\n            id\n          }\n        }\n        ...NamespaceMembershipListItemFragment_membership\n      }\n    }\n    id\n  }\n}\n\nfragment NamespaceMembershipListItemFragment_membership on NamespaceMembership {\n  metadata {\n    createdAt\n    updatedAt\n    trn\n  }\n  id\n  role {\n    name\n    id\n  }\n  resourcePath\n  member {\n    __typename\n    ... on User {\n      id\n      username\n      email\n    }\n    ... on Team {\n      id\n      name\n    }\n    ... on ServiceAccount {\n      id\n      name\n      resourcePath\n    }\n    ... on Node {\n      __isNode: __typename\n      id\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "b8babdbae79ed68e903d7b785a29bed7";

export default node;
