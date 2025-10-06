/**
 * @generated SignedSource<<0e7f6ad9255c9c85de3969e693e8589d>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type NamespaceActivityQuery$variables = {
  after?: string | null | undefined;
  before?: string | null | undefined;
  first?: number | null | undefined;
  last?: number | null | undefined;
  namespacePath?: string | null | undefined;
};
export type NamespaceActivityQuery$data = {
  readonly " $fragmentSpreads": FragmentRefs<"NamespaceActivityConnectionFragment_activity">;
};
export type NamespaceActivityQuery = {
  response: NamespaceActivityQuery$data;
  variables: NamespaceActivityQuery$variables;
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
  "name": "last"
},
v4 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "namespacePath"
},
v5 = [
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
    "kind": "Literal",
    "name": "includeNested",
    "value": true
  },
  {
    "kind": "Variable",
    "name": "last",
    "variableName": "last"
  },
  {
    "kind": "Variable",
    "name": "namespacePath",
    "variableName": "namespacePath"
  },
  {
    "kind": "Literal",
    "name": "sort",
    "value": "CREATED_DESC"
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
  "name": "__typename",
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
v10 = [
  (v8/*: any*/),
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "fullPath",
    "storageKey": null
  },
  (v9/*: any*/)
],
v11 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "resourcePath",
  "storageKey": null
},
v12 = [
  (v8/*: any*/),
  (v9/*: any*/),
  (v11/*: any*/)
],
v13 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "username",
  "storageKey": null
},
v14 = {
  "kind": "InlineFragment",
  "selections": [
    (v13/*: any*/)
  ],
  "type": "User",
  "abstractKey": null
},
v15 = [
  (v8/*: any*/)
],
v16 = {
  "kind": "InlineFragment",
  "selections": (v15/*: any*/),
  "type": "Team",
  "abstractKey": null
},
v17 = {
  "kind": "InlineFragment",
  "selections": [
    (v11/*: any*/)
  ],
  "type": "ServiceAccount",
  "abstractKey": null
},
v18 = {
  "kind": "InlineFragment",
  "selections": [
    (v6/*: any*/)
  ],
  "type": "Node",
  "abstractKey": "__isNode"
},
v19 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "registryNamespace",
  "storageKey": null
},
v20 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "system",
  "storageKey": null
},
v21 = {
  "alias": null,
  "args": null,
  "concreteType": null,
  "kind": "LinkedField",
  "name": "member",
  "plural": false,
  "selections": [
    (v7/*: any*/),
    (v14/*: any*/),
    (v17/*: any*/),
    (v16/*: any*/),
    (v18/*: any*/)
  ],
  "storageKey": null
},
v22 = [
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "previousGroupPath",
    "storageKey": null
  }
],
v23 = {
  "alias": null,
  "args": null,
  "concreteType": "User",
  "kind": "LinkedField",
  "name": "user",
  "plural": false,
  "selections": [
    (v13/*: any*/),
    (v6/*: any*/)
  ],
  "storageKey": null
},
v24 = [
  (v23/*: any*/),
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "maintainer",
    "storageKey": null
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
    "name": "NamespaceActivityQuery",
    "selections": [
      {
        "args": null,
        "kind": "FragmentSpread",
        "name": "NamespaceActivityConnectionFragment_activity"
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [
      (v2/*: any*/),
      (v3/*: any*/),
      (v0/*: any*/),
      (v1/*: any*/),
      (v4/*: any*/)
    ],
    "kind": "Operation",
    "name": "NamespaceActivityQuery",
    "selections": [
      {
        "alias": null,
        "args": (v5/*: any*/),
        "concreteType": "ActivityEventConnection",
        "kind": "LinkedField",
        "name": "activityEvents",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "ActivityEventEdge",
            "kind": "LinkedField",
            "name": "edges",
            "plural": true,
            "selections": [
              {
                "alias": null,
                "args": null,
                "concreteType": "ActivityEvent",
                "kind": "LinkedField",
                "name": "node",
                "plural": false,
                "selections": [
                  (v6/*: any*/),
                  (v7/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": null,
                    "kind": "LinkedField",
                    "name": "target",
                    "plural": false,
                    "selections": [
                      (v7/*: any*/),
                      (v6/*: any*/),
                      {
                        "kind": "InlineFragment",
                        "selections": (v10/*: any*/),
                        "type": "Workspace",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v10/*: any*/),
                        "type": "Group",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v12/*: any*/),
                        "type": "ManagedIdentity",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": null,
                            "kind": "LinkedField",
                            "name": "member",
                            "plural": false,
                            "selections": [
                              (v7/*: any*/),
                              (v14/*: any*/),
                              (v16/*: any*/),
                              (v17/*: any*/),
                              (v18/*: any*/)
                            ],
                            "storageKey": null
                          }
                        ],
                        "type": "NamespaceMembership",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "gpgKeyId",
                            "storageKey": null
                          }
                        ],
                        "type": "GPGKey",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "runStage",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "concreteType": "ManagedIdentity",
                            "kind": "LinkedField",
                            "name": "managedIdentity",
                            "plural": false,
                            "selections": [
                              (v6/*: any*/),
                              (v11/*: any*/)
                            ],
                            "storageKey": null
                          }
                        ],
                        "type": "ManagedIdentityAccessRule",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v12/*: any*/),
                        "type": "ServiceAccount",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "key",
                            "storageKey": null
                          }
                        ],
                        "type": "NamespaceVariable",
                        "abstractKey": null
                      },
                      (v16/*: any*/),
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          (v8/*: any*/),
                          (v19/*: any*/)
                        ],
                        "type": "TerraformProvider",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          (v8/*: any*/),
                          (v20/*: any*/),
                          (v19/*: any*/)
                        ],
                        "type": "TerraformModule",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
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
                            "concreteType": "TerraformModule",
                            "kind": "LinkedField",
                            "name": "module",
                            "plural": false,
                            "selections": [
                              (v8/*: any*/),
                              (v20/*: any*/),
                              (v19/*: any*/),
                              (v6/*: any*/)
                            ],
                            "storageKey": null
                          }
                        ],
                        "type": "TerraformModuleVersion",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v15/*: any*/),
                        "type": "VCSProvider",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v15/*: any*/),
                        "type": "Role",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v15/*: any*/),
                        "type": "Runner",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "hostname",
                            "storageKey": null
                          }
                        ],
                        "type": "FederatedRegistry",
                        "abstractKey": null
                      }
                    ],
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "action",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": null,
                    "kind": "LinkedField",
                    "name": "payload",
                    "plural": false,
                    "selections": [
                      (v7/*: any*/),
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          (v8/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "type",
                            "storageKey": null
                          }
                        ],
                        "type": "ActivityEventDeleteChildResourcePayload",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          (v21/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "role",
                            "storageKey": null
                          }
                        ],
                        "type": "ActivityEventCreateNamespaceMembershipPayload",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          (v21/*: any*/)
                        ],
                        "type": "ActivityEventRemoveNamespaceMembershipPayload",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v22/*: any*/),
                        "type": "ActivityEventMigrateWorkspacePayload",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v22/*: any*/),
                        "type": "ActivityEventMigrateGroupPayload",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v22/*: any*/),
                        "type": "ActivityEventMoveManagedIdentityPayload",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "prevRole",
                            "storageKey": null
                          },
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "newRole",
                            "storageKey": null
                          }
                        ],
                        "type": "ActivityEventUpdateNamespaceMembershipPayload",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v24/*: any*/),
                        "type": "ActivityEventUpdateTeamMemberPayload",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          (v23/*: any*/)
                        ],
                        "type": "ActivityEventRemoveTeamMemberPayload",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v24/*: any*/),
                        "type": "ActivityEventAddTeamMemberPayload",
                        "abstractKey": null
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
                      }
                    ],
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": null,
                    "kind": "LinkedField",
                    "name": "initiator",
                    "plural": false,
                    "selections": [
                      (v7/*: any*/),
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          (v13/*: any*/),
                          {
                            "alias": null,
                            "args": null,
                            "kind": "ScalarField",
                            "name": "email",
                            "storageKey": null
                          }
                        ],
                        "type": "User",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": [
                          (v8/*: any*/),
                          (v11/*: any*/)
                        ],
                        "type": "ServiceAccount",
                        "abstractKey": null
                      },
                      (v18/*: any*/)
                    ],
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "namespacePath",
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
        "args": (v5/*: any*/),
        "filters": [
          "namespacePath",
          "includeNested",
          "sort"
        ],
        "handle": "connection",
        "key": "NamespaceActivity_activityEvents",
        "kind": "LinkedHandle",
        "name": "activityEvents"
      }
    ]
  },
  "params": {
    "cacheID": "1bb3b5216c0fe5808f8d3dfdd271bcfa",
    "id": null,
    "metadata": {},
    "name": "NamespaceActivityQuery",
    "operationKind": "query",
    "text": "query NamespaceActivityQuery(\n  $first: Int\n  $last: Int\n  $after: String\n  $before: String\n  $namespacePath: String\n) {\n  ...NamespaceActivityConnectionFragment_activity\n}\n\nfragment ActivityEventFederatedRegistryTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on FederatedRegistry {\n      id\n      hostname\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventGPGKeyTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on GPGKey {\n      id\n      gpgKeyId\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventGroupTargetFragment_event on ActivityEvent {\n  action\n  target {\n    __typename\n    ... on Group {\n      name\n      fullPath\n      description\n    }\n    id\n  }\n  payload {\n    __typename\n    ... on ActivityEventDeleteChildResourcePayload {\n      name\n      type\n    }\n    ... on ActivityEventCreateNamespaceMembershipPayload {\n      member {\n        __typename\n        ... on User {\n          username\n        }\n        ... on ServiceAccount {\n          resourcePath\n        }\n        ... on Team {\n          name\n        }\n        ... on Node {\n          __isNode: __typename\n          id\n        }\n      }\n      role\n    }\n    ... on ActivityEventRemoveNamespaceMembershipPayload {\n      member {\n        __typename\n        ... on User {\n          username\n        }\n        ... on ServiceAccount {\n          resourcePath\n        }\n        ... on Team {\n          name\n        }\n        ... on Node {\n          __isNode: __typename\n          id\n        }\n      }\n    }\n    ... on ActivityEventMigrateGroupPayload {\n      previousGroupPath\n    }\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventListFragment_connection on ActivityEventConnection {\n  edges {\n    node {\n      id\n      target {\n        __typename\n        id\n      }\n      ...ActivityEventWorkspaceTargetFragment_event\n      ...ActivityEventGroupTargetFragment_event\n      ...ActivityEventManagedIdentityTargetFragment_event\n      ...ActivityEventNamespaceMembershipTargetFragment_event\n      ...ActivityEventGPGKeyTargetFragment_event\n      ...ActivityEventManagedIdentityAccessRuleTargetFragment_event\n      ...ActivityEventServiceAccountTargetFragment_event\n      ...ActivityEventVariableTargetFragment_event\n      ...ActivityEventRunTargetFragment_event\n      ...ActivityEventStateVersionTargetFragment_event\n      ...ActivityEventTeamTargetFragment_event\n      ...ActivityEventTerraformProviderTargetFragment_event\n      ...ActivityEventTerraformModuleTargetFragment_event\n      ...ActivityEventTerraformModuleVersionTargetFragment_event\n      ...ActivityEventVCSProviderTargetFragment_event\n      ...ActivityEventRoleTargetFragment_event\n      ...ActivityEventRunnerTargetFragment_event\n      ...ActivityEventFederatedRegistryTargetFragment_event\n    }\n  }\n}\n\nfragment ActivityEventListItemFragment_event on ActivityEvent {\n  metadata {\n    createdAt\n  }\n  id\n  initiator {\n    __typename\n    ... on User {\n      username\n      email\n    }\n    ... on ServiceAccount {\n      name\n      resourcePath\n    }\n    ... on Node {\n      __isNode: __typename\n      id\n    }\n  }\n}\n\nfragment ActivityEventManagedIdentityAccessRuleTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on ManagedIdentityAccessRule {\n      runStage\n      managedIdentity {\n        id\n        resourcePath\n      }\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventManagedIdentityTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on ManagedIdentity {\n      id\n      name\n      description\n      resourcePath\n    }\n    id\n  }\n  payload {\n    __typename\n    ... on ActivityEventMoveManagedIdentityPayload {\n      previousGroupPath\n    }\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventNamespaceMembershipTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on NamespaceMembership {\n      member {\n        __typename\n        ... on User {\n          username\n        }\n        ... on Team {\n          name\n        }\n        ... on ServiceAccount {\n          resourcePath\n        }\n        ... on Node {\n          __isNode: __typename\n          id\n        }\n      }\n    }\n    id\n  }\n  payload {\n    __typename\n    ... on ActivityEventUpdateNamespaceMembershipPayload {\n      prevRole\n      newRole\n    }\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventRoleTargetFragment_event on ActivityEvent {\n  action\n  target {\n    __typename\n    ... on Role {\n      name\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventRunTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on Run {\n      id\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventRunnerTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on Runner {\n      name\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventServiceAccountTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on ServiceAccount {\n      id\n      name\n      description\n      resourcePath\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventStateVersionTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on StateVersion {\n      id\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventTeamTargetFragment_event on ActivityEvent {\n  action\n  target {\n    __typename\n    ... on Team {\n      name\n    }\n    id\n  }\n  payload {\n    __typename\n    ... on ActivityEventUpdateTeamMemberPayload {\n      user {\n        username\n        id\n      }\n      maintainer\n    }\n    ... on ActivityEventRemoveTeamMemberPayload {\n      user {\n        username\n        id\n      }\n    }\n    ... on ActivityEventAddTeamMemberPayload {\n      user {\n        username\n        id\n      }\n      maintainer\n    }\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventTerraformModuleTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on TerraformModule {\n      name\n      system\n      registryNamespace\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventTerraformModuleVersionTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on TerraformModuleVersion {\n      version\n      module {\n        name\n        system\n        registryNamespace\n        id\n      }\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventTerraformProviderTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on TerraformProvider {\n      name\n      registryNamespace\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventVCSProviderTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on VCSProvider {\n      name\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventVariableTargetFragment_event on ActivityEvent {\n  action\n  namespacePath\n  target {\n    __typename\n    ... on NamespaceVariable {\n      key\n    }\n    id\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment ActivityEventWorkspaceTargetFragment_event on ActivityEvent {\n  action\n  target {\n    __typename\n    ... on Workspace {\n      name\n      fullPath\n      description\n    }\n    id\n  }\n  payload {\n    __typename\n    ... on ActivityEventDeleteChildResourcePayload {\n      name\n      type\n    }\n    ... on ActivityEventCreateNamespaceMembershipPayload {\n      member {\n        __typename\n        ... on User {\n          username\n        }\n        ... on ServiceAccount {\n          resourcePath\n        }\n        ... on Team {\n          name\n        }\n        ... on Node {\n          __isNode: __typename\n          id\n        }\n      }\n      role\n    }\n    ... on ActivityEventRemoveNamespaceMembershipPayload {\n      member {\n        __typename\n        ... on User {\n          username\n        }\n        ... on ServiceAccount {\n          resourcePath\n        }\n        ... on Team {\n          name\n        }\n        ... on Node {\n          __isNode: __typename\n          id\n        }\n      }\n    }\n    ... on ActivityEventMigrateWorkspacePayload {\n      previousGroupPath\n    }\n  }\n  ...ActivityEventListItemFragment_event\n}\n\nfragment NamespaceActivityConnectionFragment_activity on Query {\n  activityEvents(after: $after, before: $before, first: $first, last: $last, namespacePath: $namespacePath, includeNested: true, sort: CREATED_DESC) {\n    edges {\n      node {\n        id\n        __typename\n      }\n      cursor\n    }\n    ...ActivityEventListFragment_connection\n    pageInfo {\n      endCursor\n      hasNextPage\n      hasPreviousPage\n      startCursor\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "947695372016feb7b8409573224739a2";

export default node;
