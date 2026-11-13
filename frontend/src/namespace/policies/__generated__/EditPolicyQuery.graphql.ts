/**
 * @generated SignedSource<<5ed39b412fa3a61827f661be23d2b7a2>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type EditPolicyQuery$variables = {
  id: string;
};
export type EditPolicyQuery$data = {
  readonly node: {
    readonly policy: {
      readonly " $fragmentSpreads": FragmentRefs<"EditPolicyFragment_policy">;
    } | null | undefined;
  } | null | undefined;
};
export type EditPolicyQuery = {
  response: EditPolicyQuery$data;
  variables: EditPolicyQuery$variables;
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
  "name": "name",
  "storageKey": null
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "stage",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "enforcementLevel",
  "storageKey": null
},
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "speculativeRunEnforcementLevel",
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*:: as any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "EditPolicyQuery",
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
                  "name": "EditPolicyFragment_policy"
                }
              ],
              "type": "Policy",
              "abstractKey": null
            },
            "kind": "AliasedInlineFragmentSpread",
            "name": "policy"
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
    "name": "EditPolicyQuery",
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
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "__typename",
            "storageKey": null
          },
          (v2/*:: as any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              (v3/*:: as any*/),
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
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "packageDigest",
                    "storageKey": null
                  },
                  (v4/*:: as any*/),
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
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "verifyStateLineage",
                    "storageKey": null
                  },
                  (v4/*:: as any*/),
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
                    "name": "type",
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
                    "kind": "ScalarField",
                    "name": "pattern",
                    "storageKey": null
                  }
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "User",
                "kind": "LinkedField",
                "name": "allowedUsers",
                "plural": true,
                "selections": [
                  (v2/*:: as any*/),
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
                    "name": "username",
                    "storageKey": null
                  }
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "Team",
                "kind": "LinkedField",
                "name": "allowedTeams",
                "plural": true,
                "selections": [
                  (v2/*:: as any*/),
                  (v3/*:: as any*/)
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "ServiceAccount",
                "kind": "LinkedField",
                "name": "allowedServiceAccounts",
                "plural": true,
                "selections": [
                  (v2/*:: as any*/),
                  (v3/*:: as any*/),
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "resourcePath",
                    "storageKey": null
                  }
                ],
                "storageKey": null
              }
            ],
            "type": "Policy",
            "abstractKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "858e48c7a3be9d98c2784ce114967a62",
    "id": null,
    "metadata": {},
    "name": "EditPolicyQuery",
    "operationKind": "query",
    "text": "query EditPolicyQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ... on Policy {\n      ...EditPolicyFragment_policy\n    }\n    id\n  }\n}\n\nfragment EditPolicyFragment_policy on Policy {\n  id\n  name\n  description\n  kind\n  requiredApprovals\n  opaData {\n    packageSource\n    packageVersionConstraint\n    packageDigest\n    stage\n    enforcementLevel\n    speculativeRunEnforcementLevel\n  }\n  moduleAttestationData {\n    publicKey\n    predicateType\n    verifyStateLineage\n    stage\n    enforcementLevel\n    speculativeRunEnforcementLevel\n  }\n  scope {\n    type\n    action\n    pattern\n  }\n  allowedUsers {\n    id\n    email\n    username\n  }\n  allowedTeams {\n    id\n    name\n  }\n  allowedServiceAccounts {\n    id\n    name\n    resourcePath\n  }\n}\n"
  }
};
})();

(node as any).hash = "026459ae59352ed0852f8fe74ccaae67";

export default node;
