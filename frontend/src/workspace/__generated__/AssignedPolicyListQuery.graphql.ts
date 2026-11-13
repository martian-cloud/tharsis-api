/**
 * @generated SignedSource<<ff7dd6a1a5e7ab12a028162df042893e>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type AssignedPolicyListQuery$variables = {
  id: string;
};
export type AssignedPolicyListQuery$data = {
  readonly node: {
    readonly workspace: {
      readonly " $fragmentSpreads": FragmentRefs<"AssignedPolicyListPoliciesFragment_workspace">;
    } | null | undefined;
  } | null | undefined;
};
export type AssignedPolicyListQuery = {
  response: AssignedPolicyListQuery$data;
  variables: AssignedPolicyListQuery$variables;
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
  "name": "__typename",
  "storageKey": null
},
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v4 = [
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "trnSnapshot",
    "storageKey": null
  }
],
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "stage",
  "storageKey": null
},
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "enforcementLevel",
  "storageKey": null
},
v7 = [
  (v3/*:: as any*/)
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*:: as any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "AssignedPolicyListQuery",
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
            "fragment": {
              "kind": "InlineFragment",
              "selections": [
                {
                  "args": null,
                  "kind": "FragmentSpread",
                  "name": "AssignedPolicyListPoliciesFragment_workspace"
                }
              ],
              "type": "Workspace",
              "abstractKey": null
            },
            "kind": "AliasedInlineFragmentSpread",
            "name": "workspace"
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
    "name": "AssignedPolicyListQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*:: as any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          (v2/*:: as any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              {
                "alias": null,
                "args": null,
                "concreteType": "Policy",
                "kind": "LinkedField",
                "name": "assignedPolicies",
                "plural": true,
                "selections": [
                  (v3/*:: as any*/),
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "name",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "description",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "kind",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "createdBy",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": null,
                    "kind": "LinkedField",
                    "name": "createdByActor",
                    "plural": false,
                    "selections": [
                      (v2/*:: as any*/),
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
                              (v3/*:: as any*/)
                            ],
                            "storageKey": null
                          }
                        ],
                        "type": "UserActor",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v4/*:: as any*/),
                        "type": "ServiceAccountActor",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v4/*:: as any*/),
                        "type": "VCSProviderActor",
                        "abstractKey": null
                      },
                      {
                        "kind": "InlineFragment",
                        "selections": (v4/*:: as any*/),
                        "type": "JobActor",
                        "abstractKey": null
                      }
                    ],
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "requiredApprovals",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "OPAPolicyData",
                    "kind": "LinkedField",
                    "name": "opaData",
                    "plural": false,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "packageSource",
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "packageVersionConstraint",
                        "storageKey": null
                      },
                      (v5/*:: as any*/),
                      (v6/*:: as any*/)
                    ],
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "ModuleAttestationPolicyData",
                    "kind": "LinkedField",
                    "name": "moduleAttestationData",
                    "plural": false,
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
                      },
                      (v5/*:: as any*/),
                      (v6/*:: as any*/)
                    ],
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "PolicyScopeRule",
                    "kind": "LinkedField",
                    "name": "scope",
                    "plural": true,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "action",
                        "storageKey": null
                      }
                    ],
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "groupPath",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "User",
                    "kind": "LinkedField",
                    "name": "allowedUsers",
                    "plural": true,
                    "selections": (v7/*:: as any*/),
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "Team",
                    "kind": "LinkedField",
                    "name": "allowedTeams",
                    "plural": true,
                    "selections": (v7/*:: as any*/),
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "ServiceAccount",
                    "kind": "LinkedField",
                    "name": "allowedServiceAccounts",
                    "plural": true,
                    "selections": (v7/*:: as any*/),
                    "storageKey": null
                  }
                ],
                "storageKey": null
              }
            ],
            "type": "Workspace",
            "abstractKey": null
          },
          (v3/*:: as any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "d70087d8b057fafb6c394e8f80740b2e",
    "id": null,
    "metadata": {},
    "name": "AssignedPolicyListQuery",
    "operationKind": "query",
    "text": "query AssignedPolicyListQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ... on Workspace {\n      ...AssignedPolicyListPoliciesFragment_workspace\n    }\n    id\n  }\n}\n\nfragment ActorDisplayFragment_actor on Actor {\n  __isActor: __typename\n  type\n  principalId\n  ... on UserActor {\n    user {\n      email\n      id\n    }\n  }\n  ... on ServiceAccountActor {\n    trnSnapshot\n  }\n  ... on VCSProviderActor {\n    trnSnapshot\n  }\n  ... on JobActor {\n    trnSnapshot\n  }\n}\n\nfragment AssignedPolicyListPoliciesFragment_workspace on Workspace {\n  assignedPolicies {\n    id\n    ...PolicyCardFragment_policy\n  }\n}\n\nfragment PolicyCardFragment_policy on Policy {\n  id\n  name\n  description\n  kind\n  createdBy\n  createdByActor {\n    __typename\n    ...ActorDisplayFragment_actor\n  }\n  requiredApprovals\n  opaData {\n    packageSource\n    packageVersionConstraint\n    stage\n    enforcementLevel\n  }\n  moduleAttestationData {\n    publicKey\n    predicateType\n    stage\n    enforcementLevel\n  }\n  scope {\n    action\n  }\n  groupPath\n  allowedUsers {\n    id\n  }\n  allowedTeams {\n    id\n  }\n  allowedServiceAccounts {\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "7c0f4e77f9dd88b4fe616486e2a0b13e";

export default node;
