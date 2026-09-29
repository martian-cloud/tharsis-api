/**
 * @generated SignedSource<<a4f7ae22bc4eed5956ace5d29b5dd3cd>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type AssignedManagedIdentityListQuery$variables = {
  id: string;
};
export type AssignedManagedIdentityListQuery$data = {
  readonly node: {
    readonly " $fragmentSpreads": FragmentRefs<"AssignedManagedIdentityListAssignedFragment_assignedManagedIdentities">;
  } | null | undefined;
};
export type AssignedManagedIdentityListQuery = {
  response: AssignedManagedIdentityListQuery$data;
  variables: AssignedManagedIdentityListQuery$variables;
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
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "AssignedManagedIdentityListQuery",
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
                "args": null,
                "kind": "FragmentSpread",
                "name": "AssignedManagedIdentityListAssignedFragment_assignedManagedIdentities"
              }
            ],
            "type": "Workspace",
            "abstractKey": null
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
    "name": "AssignedManagedIdentityListQuery",
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
                "args": [
                  {
                    "kind": "Literal",
                    "name": "first",
                    "value": 1
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
                          (v2/*: any*/)
                        ],
                        "storageKey": null
                      }
                    ],
                    "storageKey": null
                  }
                ],
                "storageKey": "managedIdentities(first:1,includeInherited:true)"
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "ManagedIdentity",
                "kind": "LinkedField",
                "name": "assignedManagedIdentities",
                "plural": true,
                "selections": [
                  (v2/*: any*/),
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
                    "name": "resourcePath",
                    "storageKey": null
                  }
                ],
                "storageKey": null
              }
            ],
            "type": "Workspace",
            "abstractKey": null
          },
          (v2/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "708292ff28f84122be4c787766007a33",
    "id": null,
    "metadata": {},
    "name": "AssignedManagedIdentityListQuery",
    "operationKind": "query",
    "text": "query AssignedManagedIdentityListQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ... on Workspace {\n      ...AssignedManagedIdentityListAssignedFragment_assignedManagedIdentities\n    }\n    id\n  }\n}\n\nfragment AssignedManagedIdentityListAssignedFragment_assignedManagedIdentities on Workspace {\n  managedIdentities(includeInherited: true, first: 1) {\n    edges {\n      node {\n        id\n      }\n    }\n  }\n  assignedManagedIdentities {\n    id\n    ...AssignedManagedIdentityListItemFragment_managedIdentity\n  }\n}\n\nfragment AssignedManagedIdentityListItemFragment_managedIdentity on ManagedIdentity {\n  metadata {\n    updatedAt\n  }\n  id\n  name\n  description\n  type\n  resourcePath\n}\n"
  }
};
})();

(node as any).hash = "32579b048b741923349d6863dd3a7cf7";

export default node;
