/**
 * @generated SignedSource<<a378187db53b2f9a3e8adcbbe2b3e66d>>
 * @lightSyntaxTransform
 * @nogrep
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
    readonly " $fragmentSpreads": FragmentRefs<"WorkspaceRoleBindingRoleFragment_workspace">;
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
          "selections": (v3/*: any*/),
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
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "WorkspaceRoleBindingQuery",
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
            "name": "WorkspaceRoleBindingRoleFragment_workspace"
          }
        ],
        "storageKey": null
      },
      (v4/*: any*/)
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "WorkspaceRoleBindingQuery",
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
                "concreteType": "WorkspaceRoleBinding",
                "kind": "LinkedField",
                "name": "roleBinding",
                "plural": false,
                "selections": [
                  (v2/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "Role",
                    "kind": "LinkedField",
                    "name": "role",
                    "plural": false,
                    "selections": (v3/*: any*/),
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
      },
      (v4/*: any*/)
    ]
  },
  "params": {
    "cacheID": "ca46c6982af8758638f77fbbfb9a68a4",
    "id": null,
    "metadata": {},
    "name": "WorkspaceRoleBindingQuery",
    "operationKind": "query",
    "text": "query WorkspaceRoleBindingQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ...WorkspaceRoleBindingRoleFragment_workspace\n    id\n  }\n  roles(first: 5) {\n    edges {\n      node {\n        id\n        name\n        description\n      }\n    }\n  }\n}\n\nfragment WorkspaceRoleBindingRoleFragment_workspace on Workspace {\n  roleBinding {\n    id\n    role {\n      id\n      name\n      description\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "23d1700e13c5b21b2f49635ef76a450f";

export default node;
