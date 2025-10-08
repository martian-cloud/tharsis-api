/**
 * @generated SignedSource<<410d633a833ef58f9d397cf8299e4912>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type JobType = "apply" | "plan" | "%future added value";
export type ManagedIdentityDetailsQuery$variables = {
  after?: string | null | undefined;
  before?: string | null | undefined;
  first: number;
  id: string;
  last?: number | null | undefined;
};
export type ManagedIdentityDetailsQuery$data = {
  readonly managedIdentity: {
    readonly accessRules: ReadonlyArray<{
      readonly allowedServiceAccounts: ReadonlyArray<{
        readonly id: string;
        readonly name: string;
        readonly resourcePath: string;
      }> | null | undefined;
      readonly allowedTeams: ReadonlyArray<{
        readonly id: string;
        readonly name: string;
      }> | null | undefined;
      readonly allowedUsers: ReadonlyArray<{
        readonly email: string;
        readonly id: string;
        readonly username: string;
      }> | null | undefined;
      readonly id: string;
      readonly runStage: JobType;
    }>;
    readonly data: string;
    readonly description: string;
    readonly groupPath: string;
    readonly id: string;
    readonly isAlias: boolean;
    readonly metadata: {
      readonly trn: string;
    };
    readonly name: string;
    readonly type: string;
    readonly " $fragmentSpreads": FragmentRefs<"ManagedIdentityAliasesFragment_managedIdentity" | "ManagedIdentityRulesFragment_managedIdentity" | "MoveManagedIdentityDialogFragment_managedIdentity">;
  } | null | undefined;
};
export type ManagedIdentityDetailsQuery = {
  response: ManagedIdentityDetailsQuery$data;
  variables: ManagedIdentityDetailsQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "after"
},
v1 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "before"
},
v2 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "first"
},
v3 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "id"
},
v4 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "last"
},
v5 = [
  {
    "kind": "Variable",
    "name": "id",
    "variableName": "id"
  }
],
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "isAlias",
  "storageKey": null
},
v8 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "name",
  "storageKey": null
},
v9 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "description",
  "storageKey": null
},
v10 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "type",
  "storageKey": null
},
v11 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "data",
  "storageKey": null
},
v12 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "groupPath",
  "storageKey": null
},
v13 = {
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
      "name": "trn",
      "storageKey": null
    }
  ],
  "storageKey": null
},
v14 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "runStage",
  "storageKey": null
},
v15 = {
  "alias": null,
  "args": null,
  "concreteType": "User",
  "kind": "LinkedField",
  "name": "allowedUsers",
  "plural": true,
  "selections": [
    (v6/*: any*/),
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
    }
  ],
  "storageKey": null
},
v16 = {
  "alias": null,
  "args": null,
  "concreteType": "Team",
  "kind": "LinkedField",
  "name": "allowedTeams",
  "plural": true,
  "selections": [
    (v6/*: any*/),
    (v8/*: any*/)
  ],
  "storageKey": null
},
v17 = {
  "alias": null,
  "args": null,
  "concreteType": "ServiceAccount",
  "kind": "LinkedField",
  "name": "allowedServiceAccounts",
  "plural": true,
  "selections": [
    (v6/*: any*/),
    (v8/*: any*/),
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "resourcePath",
      "storageKey": null
    }
  ],
  "storageKey": null
},
v18 = [
  {
    "kind": "Variable",
    "name": "after",
    "variableName": "after"
  },
  {
    "kind": "Variable",
    "name": "before",
    "variableName": "before"
  },
  {
    "kind": "Variable",
    "name": "first",
    "variableName": "first"
  },
  {
    "kind": "Variable",
    "name": "last",
    "variableName": "last"
  }
];
return {
  "fragment": {
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/),
      (v2/*: any*/),
      (v3/*: any*/),
      (v4/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "ManagedIdentityDetailsQuery",
    "selections": [
      {
        "alias": null,
        "args": (v5/*: any*/),
        "concreteType": "ManagedIdentity",
        "kind": "LinkedField",
        "name": "managedIdentity",
        "plural": false,
        "selections": [
          (v6/*: any*/),
          (v7/*: any*/),
          (v8/*: any*/),
          (v9/*: any*/),
          (v10/*: any*/),
          (v11/*: any*/),
          (v12/*: any*/),
          (v13/*: any*/),
          {
            "alias": null,
            "args": null,
            "concreteType": "ManagedIdentityAccessRule",
            "kind": "LinkedField",
            "name": "accessRules",
            "plural": true,
            "selections": [
              (v6/*: any*/),
              (v14/*: any*/),
              (v15/*: any*/),
              (v16/*: any*/),
              (v17/*: any*/)
            ],
            "storageKey": null
          },
          {
            "args": null,
            "kind": "FragmentSpread",
            "name": "ManagedIdentityAliasesFragment_managedIdentity"
          },
          {
            "args": null,
            "kind": "FragmentSpread",
            "name": "ManagedIdentityRulesFragment_managedIdentity"
          },
          {
            "args": null,
            "kind": "FragmentSpread",
            "name": "MoveManagedIdentityDialogFragment_managedIdentity"
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
    "argumentDefinitions": [
      (v3/*: any*/),
      (v2/*: any*/),
      (v0/*: any*/),
      (v4/*: any*/),
      (v1/*: any*/)
    ],
    "kind": "Operation",
    "name": "ManagedIdentityDetailsQuery",
    "selections": [
      {
        "alias": null,
        "args": (v5/*: any*/),
        "concreteType": "ManagedIdentity",
        "kind": "LinkedField",
        "name": "managedIdentity",
        "plural": false,
        "selections": [
          (v6/*: any*/),
          (v7/*: any*/),
          (v8/*: any*/),
          (v9/*: any*/),
          (v10/*: any*/),
          (v11/*: any*/),
          (v12/*: any*/),
          (v13/*: any*/),
          {
            "alias": null,
            "args": null,
            "concreteType": "ManagedIdentityAccessRule",
            "kind": "LinkedField",
            "name": "accessRules",
            "plural": true,
            "selections": [
              (v6/*: any*/),
              (v14/*: any*/),
              (v15/*: any*/),
              (v16/*: any*/),
              (v17/*: any*/),
              (v10/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "ManagedIdentityAccessRuleModuleAttestationPolicy",
                "kind": "LinkedField",
                "name": "moduleAttestationPolicies",
                "plural": true,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "publicKey",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "predicateType",
                    "storageKey": null
                  }
                ],
                "storageKey": null
              }
            ],
            "storageKey": null
          },
          {
            "alias": null,
            "args": (v18/*: any*/),
            "concreteType": "ManagedIdentityConnection",
            "kind": "LinkedField",
            "name": "aliases",
            "plural": false,
            "selections": [
              {
                "alias": null,
                "args": null,
                "concreteType": "ManagedIdentityEdge",
                "kind": "LinkedField",
                "name": "edges",
                "plural": true,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "ManagedIdentity",
                    "kind": "LinkedField",
                    "name": "node",
                    "plural": false,
                    "selections": [
                      (v6/*: any*/),
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
                            "name": "updatedAt",
                            "storageKey": null
                          }
                        ],
                        "storageKey": null
                      },
                      (v8/*: any*/),
                      (v9/*: any*/),
                      (v10/*: any*/),
                      (v12/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "__typename",
                        "storageKey": null
                      }
                    ],
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "cursor",
                    "storageKey": null
                  }
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "PageInfo",
                "kind": "LinkedField",
                "name": "pageInfo",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "endCursor",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "hasNextPage",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "hasPreviousPage",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "startCursor",
                    "storageKey": null
                  }
                ],
                "storageKey": null
              }
            ],
            "storageKey": null
          },
          {
            "alias": null,
            "args": (v18/*: any*/),
            "filters": null,
            "handle": "connection",
            "key": "ManagedIdentityAliasesList_aliases",
            "kind": "LinkedHandle",
            "name": "aliases"
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "df97a07d83fadc24a520b28f3ad13b9d",
    "id": null,
    "metadata": {},
    "name": "ManagedIdentityDetailsQuery",
    "operationKind": "query",
    "text": "query ManagedIdentityDetailsQuery(\n  $id: String!\n  $first: Int!\n  $after: String\n  $last: Int\n  $before: String\n) {\n  managedIdentity(id: $id) {\n    id\n    isAlias\n    name\n    description\n    type\n    data\n    groupPath\n    metadata {\n      trn\n    }\n    accessRules {\n      id\n      runStage\n      allowedUsers {\n        id\n        username\n        email\n      }\n      allowedTeams {\n        id\n        name\n      }\n      allowedServiceAccounts {\n        id\n        name\n        resourcePath\n      }\n    }\n    ...ManagedIdentityAliasesFragment_managedIdentity\n    ...ManagedIdentityRulesFragment_managedIdentity\n    ...MoveManagedIdentityDialogFragment_managedIdentity\n  }\n}\n\nfragment ManagedIdentityAliasesFragment_managedIdentity on ManagedIdentity {\n  ...ManagedIdentityAliasesListFragment_managedIdentity\n  ...NewManagedIdentityAliasDialogFragment_managedIdentity\n}\n\nfragment ManagedIdentityAliasesListFragment_managedIdentity on ManagedIdentity {\n  id\n  aliases(first: $first, last: $last, after: $after, before: $before) {\n    edges {\n      node {\n        id\n        ...ManagedIdentityAliasesListItemFragment_managedIdentity\n        __typename\n      }\n      cursor\n    }\n    pageInfo {\n      endCursor\n      hasNextPage\n      hasPreviousPage\n      startCursor\n    }\n  }\n}\n\nfragment ManagedIdentityAliasesListItemFragment_managedIdentity on ManagedIdentity {\n  metadata {\n    updatedAt\n  }\n  id\n  name\n  description\n  type\n  groupPath\n}\n\nfragment ManagedIdentityRulesFragment_managedIdentity on ManagedIdentity {\n  id\n  isAlias\n  accessRules {\n    id\n    type\n    runStage\n    moduleAttestationPolicies {\n      publicKey\n      predicateType\n    }\n    allowedUsers {\n      id\n      username\n      email\n    }\n    allowedTeams {\n      id\n      name\n    }\n    allowedServiceAccounts {\n      id\n      name\n      resourcePath\n    }\n  }\n}\n\nfragment MoveManagedIdentityDialogFragment_managedIdentity on ManagedIdentity {\n  id\n  name\n  groupPath\n}\n\nfragment NewManagedIdentityAliasDialogFragment_managedIdentity on ManagedIdentity {\n  id\n  groupPath\n}\n"
  }
};
})();

(node as any).hash = "28bb7c044ba6d466d180238f48bf43e2";

export default node;
