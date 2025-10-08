/**
 * @generated SignedSource<<3b218c938f4a0cb74589dbab2a94ff2a>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type CreateManagedIdentityAliasInput = {
  aliasSourceId?: string | null | undefined;
  aliasSourcePath?: string | null | undefined;
  clientMutationId?: string | null | undefined;
  groupId?: string | null | undefined;
  groupPath?: string | null | undefined;
  name: string;
};
export type NewManagedIdentityAliasDialogMutation$variables = {
  connections: ReadonlyArray<string>;
  input: CreateManagedIdentityAliasInput;
};
export type NewManagedIdentityAliasDialogMutation$data = {
  readonly createManagedIdentityAlias: {
    readonly managedIdentity: {
      readonly id: string;
      readonly " $fragmentSpreads": FragmentRefs<"ManagedIdentityAliasesListItemFragment_managedIdentity">;
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type NewManagedIdentityAliasDialogMutation = {
  response: NewManagedIdentityAliasDialogMutation$data;
  variables: NewManagedIdentityAliasDialogMutation$variables;
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
    "name": "NewManagedIdentityAliasDialogMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "CreateManagedIdentityAliasPayload",
        "kind": "LinkedField",
        "name": "createManagedIdentityAlias",
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
                "name": "ManagedIdentityAliasesListItemFragment_managedIdentity"
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
    "name": "NewManagedIdentityAliasDialogMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "CreateManagedIdentityAliasPayload",
        "kind": "LinkedField",
        "name": "createManagedIdentityAlias",
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
    "cacheID": "4acf572807654fee755ccd5018ae211a",
    "id": null,
    "metadata": {},
    "name": "NewManagedIdentityAliasDialogMutation",
    "operationKind": "mutation",
    "text": "mutation NewManagedIdentityAliasDialogMutation(\n  $input: CreateManagedIdentityAliasInput!\n) {\n  createManagedIdentityAlias(input: $input) {\n    managedIdentity {\n      id\n      ...ManagedIdentityAliasesListItemFragment_managedIdentity\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment ManagedIdentityAliasesListItemFragment_managedIdentity on ManagedIdentity {\n  metadata {\n    updatedAt\n  }\n  id\n  name\n  description\n  type\n  groupPath\n}\n"
  }
};
})();

(node as any).hash = "cb0211776f5982715aa35967c834004d";

export default node;
