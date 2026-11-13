/**
 * @generated SignedSource<<e454db45c10d85cd655f3f67ee19b8e7>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type WorkspaceRoleBindingQuery$variables = {
  id: string;
};
export type WorkspaceRoleBindingQuery$data = {
  readonly node: {
    readonly workspace: {
      readonly " $fragmentSpreads": FragmentRefs<"WorkspaceRoleBindingRoleFragment_workspace">;
    } | null | undefined;
  } | null | undefined;
  readonly roles: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly description: string;
        readonly id: string;
        readonly name: string;
      } | null | undefined;
    } | null | undefined> | null | undefined;
  };
};
export type WorkspaceRoleBindingQuery = {
  response: WorkspaceRoleBindingQuery$data;
  variables: WorkspaceRoleBindingQuery$variables;
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
v3 = [
  (v2/*:: as any*/),
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
  }
],
v4 = {
  "alias": null,
  "args": [
    {
      "kind": "Literal",
      "name": "first",
      "value": 5
    }
  ],
  "concreteType": "RoleConnection",
  "kind": "LinkedField",
  "name": "roles",
  "plural": false,
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "RoleEdge",
      "kind": "LinkedField",
      "name": "edges",
      "plural": true,
      "selections": [
        {
          "alias": null,
          "args": null,
          "concreteType": "Role",
          "kind": "LinkedField",
          "name": "node",
          "plural": false,
          "selections": (v3/*:: as any*/),
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "storageKey": "roles(first:5)"
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*:: as any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "WorkspaceRoleBindingQuery",
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
                  "name": "WorkspaceRoleBindingRoleFragment_workspace"
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
      },
      (v4/*:: as any*/)
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*:: as any*/),
    "kind": "Operation",
    "name": "WorkspaceRoleBindingQuery",
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
          {
            "kind": "InlineFragment",
            "selections": [
              {
                "alias": null,
                "args": null,
                "concreteType": "WorkspaceRoleBinding",
                "kind": "LinkedField",
                "name": "roleBinding",
                "plural": false,
                "selections": [
                  (v2/*:: as any*/),
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "Role",
                    "kind": "LinkedField",
                    "name": "role",
                    "plural": false,
                    "selections": (v3/*:: as any*/),
                    "storageKey": null
                  }
                ],
                "storageKey": null
              }
            ],
            "type": "Workspace",
            "abstractKey": null
          },
          (v2/*:: as any*/)
        ],
        "storageKey": null
      },
      (v4/*:: as any*/)
    ]
  },
  "params": {
    "cacheID": "bfd786dd482a7574a791583682ae2dcd",
    "id": null,
    "metadata": {},
    "name": "WorkspaceRoleBindingQuery",
    "operationKind": "query",
    "text": "query WorkspaceRoleBindingQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ... on Workspace {\n      ...WorkspaceRoleBindingRoleFragment_workspace\n    }\n    id\n  }\n  roles(first: 5) {\n    edges {\n      node {\n        id\n        name\n        description\n      }\n    }\n  }\n}\n\nfragment WorkspaceRoleBindingRoleFragment_workspace on Workspace {\n  roleBinding {\n    id\n    role {\n      id\n      name\n      description\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "f7c677753ec93b9dd8b42413077a1926";

export default node;
