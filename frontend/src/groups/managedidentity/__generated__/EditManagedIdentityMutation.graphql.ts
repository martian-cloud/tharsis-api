/**
 * @generated SignedSource<<dfb2da560964589ee944b3b430aa4a01>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type UpdateManagedIdentityInput = {
  clientMutationId?: string | null | undefined;
  data: string;
  description: string;
  id: string;
  metadata?: ResourceMetadataInput | null | undefined;
};
export type ResourceMetadataInput = {
  version: string;
};
export type EditManagedIdentityMutation$variables = {
  input: UpdateManagedIdentityInput;
};
export type EditManagedIdentityMutation$data = {
  readonly updateManagedIdentity: {
    readonly managedIdentity: {
      readonly data: string;
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
export type EditManagedIdentityMutation = {
  response: EditManagedIdentityMutation$data;
  variables: EditManagedIdentityMutation$variables;
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
  "name": "id",
  "storageKey": null
},
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "data",
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
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "EditManagedIdentityMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "UpdateManagedIdentityPayload",
        "kind": "LinkedField",
        "name": "updateManagedIdentity",
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
              (v2/*: any*/),
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
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "EditManagedIdentityMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "UpdateManagedIdentityPayload",
        "kind": "LinkedField",
        "name": "updateManagedIdentity",
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
              (v2/*: any*/),
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
          (v5/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "62d6fb13196fc2b6eac8f97a26964b4c",
    "id": null,
    "metadata": {},
    "name": "EditManagedIdentityMutation",
    "operationKind": "mutation",
    "text": "mutation EditManagedIdentityMutation(\n  $input: UpdateManagedIdentityInput!\n) {\n  updateManagedIdentity(input: $input) {\n    managedIdentity {\n      id\n      data\n      ...ManagedIdentityListItemFragment_managedIdentity\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment ManagedIdentityListItemFragment_managedIdentity on ManagedIdentity {\n  metadata {\n    updatedAt\n  }\n  id\n  isAlias\n  name\n  description\n  type\n  resourcePath\n  groupPath\n}\n"
  }
};
})();

(node as any).hash = "87316e6ecf2b35dcebb9b43d0b3e589d";

export default node;
