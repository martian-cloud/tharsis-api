/**
 * @generated SignedSource<<d290d4957b318c9f1c9ba1fab300e50c>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type AssignManagedIdentityInput = {
  clientMutationId?: string | null | undefined;
  managedIdentityId?: string | null | undefined;
  managedIdentityPath?: string | null | undefined;
  workspaceId?: string | null | undefined;
  workspacePath?: string | null | undefined;
};
export type AssignedManagedIdentityListUnassignMutation$variables = {
  input: AssignManagedIdentityInput;
};
export type AssignedManagedIdentityListUnassignMutation$data = {
  readonly unassignManagedIdentity: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly workspace: {
      readonly " $fragmentSpreads": FragmentRefs<"AssignedManagedIdentityListFragment_assignedManagedIdentities">;
    } | null | undefined;
  };
};
export type AssignedManagedIdentityListUnassignMutation = {
  response: AssignedManagedIdentityListUnassignMutation$data;
  variables: AssignedManagedIdentityListUnassignMutation$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "input"
  }
],
v1 = [
  {
    "kind": "Variable",
    "name": "input",
    "variableName": "input"
  }
],
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "type",
  "storageKey": null
},
v3 = {
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
    (v2/*: any*/)
  ],
  "storageKey": null
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "AssignedManagedIdentityListUnassignMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "AssignManagedIdentityPayload",
        "kind": "LinkedField",
        "name": "unassignManagedIdentity",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "Workspace",
            "kind": "LinkedField",
            "name": "workspace",
            "plural": false,
            "selections": [
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "AssignedManagedIdentityListFragment_assignedManagedIdentities"
              }
            ],
            "storageKey": null
          },
          (v3/*: any*/)
        ],
        "storageKey": null
      }
    ],
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "AssignedManagedIdentityListUnassignMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "AssignManagedIdentityPayload",
        "kind": "LinkedField",
        "name": "unassignManagedIdentity",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "Workspace",
            "kind": "LinkedField",
            "name": "workspace",
            "plural": false,
            "selections": [
              (v4/*: any*/),
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "fullPath",
                "storageKey": null
              },
              {
                "alias": null,
                "args": [
                  {
                    "kind": "Literal",
                    "name": "first",
                    "value": 0
                  },
                  {
                    "kind": "Literal",
                    "name": "includeInherited",
                    "value": true
                  }
                ],
                "concreteType": "ManagedIdentityConnection",
                "kind": "LinkedField",
                "name": "managedIdentities",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "totalCount",
                    "storageKey": null
                  }
                ],
                "storageKey": "managedIdentities(first:0,includeInherited:true)"
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "ManagedIdentity",
                "kind": "LinkedField",
                "name": "assignedManagedIdentities",
                "plural": true,
                "selections": [
                  (v4/*: any*/),
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
                  (v2/*: any*/),
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
            "storageKey": null
          },
          (v3/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "d634ca6ab5258fb916fe06e7799e328a",
    "id": null,
    "metadata": {},
    "name": "AssignedManagedIdentityListUnassignMutation",
    "operationKind": "mutation",
    "text": "mutation AssignedManagedIdentityListUnassignMutation(\n  $input: AssignManagedIdentityInput!\n) {\n  unassignManagedIdentity(input: $input) {\n    workspace {\n      ...AssignedManagedIdentityListFragment_assignedManagedIdentities\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment AssignedManagedIdentityListFragment_assignedManagedIdentities on Workspace {\n  id\n  fullPath\n  managedIdentities(includeInherited: true, first: 0) {\n    totalCount\n  }\n  assignedManagedIdentities {\n    id\n    ...AssignedManagedIdentityListItemFragment_managedIdentity\n  }\n}\n\nfragment AssignedManagedIdentityListItemFragment_managedIdentity on ManagedIdentity {\n  metadata {\n    updatedAt\n  }\n  id\n  name\n  description\n  type\n  resourcePath\n}\n"
  }
};
})();

(node as any).hash = "331e0d1cd0a18d7fabc8ba461e5d2d1c";

export default node;
