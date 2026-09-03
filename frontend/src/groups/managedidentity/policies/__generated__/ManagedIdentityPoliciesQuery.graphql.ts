/**
 * @generated SignedSource<<c5934f4d7546cf3addd43de912d29646>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ManagedIdentityPoliciesQuery$variables = {
  id: string;
};
export type ManagedIdentityPoliciesQuery$data = {
  readonly node: {
    readonly " $fragmentSpreads": FragmentRefs<"ManagedIdentityPoliciesPoliciesFragment_managedIdentity">;
  } | null | undefined;
};
export type ManagedIdentityPoliciesQuery = {
  response: ManagedIdentityPoliciesQuery$data;
  variables: ManagedIdentityPoliciesQuery$variables;
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
  "name": "stage",
  "storageKey": null
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "enforcementLevel",
  "storageKey": null
},
v5 = [
  (v2/*: any*/)
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "ManagedIdentityPoliciesQuery",
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
            "args": null,
            "kind": "FragmentSpread",
            "name": "ManagedIdentityPoliciesPoliciesFragment_managedIdentity"
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
    "name": "ManagedIdentityPoliciesQuery",
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
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "__typename",
            "storageKey": null
          },
          {
            "kind": "InlineFragment",
            "selections": [
              {
                "alias": null,
                "args": null,
                "concreteType": "Policy",
                "kind": "LinkedField",
                "name": "referencingPolicies",
                "plural": true,
                "selections": [
                  (v2/*: any*/),
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
                      (v3/*: any*/),
                      (v4/*: any*/)
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
                      (v3/*: any*/),
                      (v4/*: any*/)
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
                    "selections": (v5/*: any*/),
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "Team",
                    "kind": "LinkedField",
                    "name": "allowedTeams",
                    "plural": true,
                    "selections": (v5/*: any*/),
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "ServiceAccount",
                    "kind": "LinkedField",
                    "name": "allowedServiceAccounts",
                    "plural": true,
                    "selections": (v5/*: any*/),
                    "storageKey": null
                  }
                ],
                "storageKey": null
              }
            ],
            "type": "ManagedIdentity",
            "abstractKey": null
          },
          (v2/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "978dbf441f4c5bc2d3afd0653f2c68d3",
    "id": null,
    "metadata": {},
    "name": "ManagedIdentityPoliciesQuery",
    "operationKind": "query",
    "text": "query ManagedIdentityPoliciesQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ...ManagedIdentityPoliciesPoliciesFragment_managedIdentity\n    id\n  }\n}\n\nfragment ManagedIdentityPoliciesPoliciesFragment_managedIdentity on ManagedIdentity {\n  referencingPolicies {\n    id\n    ...PolicyCardFragment_policy\n  }\n}\n\nfragment PolicyCardFragment_policy on Policy {\n  id\n  name\n  description\n  kind\n  createdBy\n  requiredApprovals\n  opaData {\n    packageSource\n    packageVersionConstraint\n    stage\n    enforcementLevel\n  }\n  moduleAttestationData {\n    publicKey\n    predicateType\n    stage\n    enforcementLevel\n  }\n  scope {\n    action\n  }\n  groupPath\n  allowedUsers {\n    id\n  }\n  allowedTeams {\n    id\n  }\n  allowedServiceAccounts {\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "072cbdb317845aada1f036afbb994a49";

export default node;
