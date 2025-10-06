/**
 * @generated SignedSource<<628055d2e0067c376835ad8b4f756916>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type JobType = "apply" | "plan" | "%future added value";
export type ManagedIdentityAccessRuleType = "eligible_principals" | "module_attestation" | "%future added value";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type CreateManagedIdentityInput = {
  accessRules?: ReadonlyArray<ManagedIdentityAccessRuleInput> | null | undefined;
  clientMutationId?: string | null | undefined;
  data: string;
  description: string;
  groupId?: string | null | undefined;
  groupPath?: string | null | undefined;
  name: string;
  type: string;
};
export type ManagedIdentityAccessRuleInput = {
  allowedServiceAccountIds?: ReadonlyArray<string> | null | undefined;
  allowedServiceAccounts?: ReadonlyArray<string> | null | undefined;
  allowedTeamIds?: ReadonlyArray<string> | null | undefined;
  allowedTeams?: ReadonlyArray<string> | null | undefined;
  allowedUserIds?: ReadonlyArray<string> | null | undefined;
  allowedUsers?: ReadonlyArray<string> | null | undefined;
  moduleAttestationPolicies?: ReadonlyArray<ManagedIdentityAccessRuleModuleAttestationPolicyInput> | null | undefined;
  runStage: JobType;
  type: ManagedIdentityAccessRuleType;
  verifyStateLineage?: boolean | null | undefined;
};
export type ManagedIdentityAccessRuleModuleAttestationPolicyInput = {
  predicateType?: string | null | undefined;
  publicKey: string;
};
export type NewManagedIdentityMutation$variables = {
  connections: ReadonlyArray<string>;
  input: CreateManagedIdentityInput;
};
export type NewManagedIdentityMutation$data = {
  readonly createManagedIdentity: {
    readonly managedIdentity: {
      readonly id: string;
      readonly " $fragmentSpreads": FragmentRefs<"ManagedIdentityListItemFragment_managedIdentity">;
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type NewManagedIdentityMutation = {
  response: NewManagedIdentityMutation$data;
  variables: NewManagedIdentityMutation$variables;
};

const node: ConcreteRequest = (function(){
var v0 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "connections"
},
v1 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "input"
},
v2 = [
  {
    "kind": "Variable",
    "name": "input",
    "variableName": "input"
  }
],
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "type",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "concreteType": "Problem",
  "kind": "LinkedField",
  "name": "problems",
  "plural": true,
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "message",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "field",
      "storageKey": null
    },
    (v4/*: any*/)
  ],
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "NewManagedIdentityMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "CreateManagedIdentityPayload",
        "kind": "LinkedField",
        "name": "createManagedIdentity",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "ManagedIdentity",
            "kind": "LinkedField",
            "name": "managedIdentity",
            "plural": false,
            "selections": [
              (v3/*: any*/),
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "ManagedIdentityListItemFragment_managedIdentity"
              }
            ],
            "storageKey": null
          },
          (v5/*: any*/)
        ],
        "storageKey": null
      }
    ],
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [
      (v1/*: any*/),
      (v0/*: any*/)
    ],
    "kind": "Operation",
    "name": "NewManagedIdentityMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "CreateManagedIdentityPayload",
        "kind": "LinkedField",
        "name": "createManagedIdentity",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "ManagedIdentity",
            "kind": "LinkedField",
            "name": "managedIdentity",
            "plural": false,
            "selections": [
              (v3/*: any*/),
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
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "isAlias",
                "storageKey": null
              },
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
              (v4/*: any*/),
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "resourcePath",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "groupPath",
                "storageKey": null
              }
            ],
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "filters": null,
            "handle": "prependNode",
            "key": "",
            "kind": "LinkedHandle",
            "name": "managedIdentity",
            "handleArgs": [
              {
                "kind": "Variable",
                "name": "connections",
                "variableName": "connections"
              },
              {
                "kind": "Literal",
                "name": "edgeTypeName",
                "value": "ManagedIdentityEdge"
              }
            ]
          },
          (v5/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "ac81dfc7d133be9b81273a9946f91b3f",
    "id": null,
    "metadata": {},
    "name": "NewManagedIdentityMutation",
    "operationKind": "mutation",
    "text": "mutation NewManagedIdentityMutation(\n  $input: CreateManagedIdentityInput!\n) {\n  createManagedIdentity(input: $input) {\n    managedIdentity {\n      id\n      ...ManagedIdentityListItemFragment_managedIdentity\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment ManagedIdentityListItemFragment_managedIdentity on ManagedIdentity {\n  metadata {\n    updatedAt\n  }\n  id\n  isAlias\n  name\n  description\n  type\n  resourcePath\n  groupPath\n}\n"
  }
};
})();

(node as any).hash = "39e88bab7c17e8acde5042edd3005976";

export default node;
