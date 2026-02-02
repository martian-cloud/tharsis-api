/**
 * @generated SignedSource<<fd645538d5f259adda23829eda1e50e0>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type UniversalSearchQuery$variables = {
  query: string;
};
export type UniversalSearchQuery$data = {
  readonly search: {
    readonly results: ReadonlyArray<{
      readonly __typename: "Group";
      readonly fullPath: string;
      readonly id: string;
      readonly name: string;
    } | {
      readonly __typename: "NamespaceFavorite";
      readonly id: string;
      readonly namespace: {
        readonly __typename: "Group";
        readonly fullPath: string;
      } | {
        readonly __typename: "Workspace";
        readonly fullPath: string;
      } | {
        // This will never be '%other', but we need some
        // value in case none of the concrete values match.
        readonly __typename: "%other";
      };
    } | {
      readonly __typename: "Team";
      readonly id: string;
      readonly name: string;
    } | {
      readonly __typename: "TerraformModule";
      readonly groupPath: string;
      readonly id: string;
      readonly name: string;
      readonly registryNamespace: string;
      readonly system: string;
    } | {
      readonly __typename: "TerraformProvider";
      readonly groupPath: string;
      readonly id: string;
      readonly name: string;
      readonly registryNamespace: string;
    } | {
      readonly __typename: "Workspace";
      readonly fullPath: string;
      readonly id: string;
      readonly name: string;
    } | {
      // This will never be '%other', but we need some
      // value in case none of the concrete values match.
      readonly __typename: "%other";
    }>;
  };
};
export type UniversalSearchQuery = {
  response: UniversalSearchQuery$data;
  variables: UniversalSearchQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "query"
  }
],
v1 = [
  {
    "kind": "Variable",
    "name": "query",
    "variableName": "query"
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
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "fullPath",
  "storageKey": null
},
v5 = [
  (v4/*: any*/)
],
v6 = {
  "kind": "InlineFragment",
  "selections": (v5/*: any*/),
  "type": "Group",
  "abstractKey": null
},
v7 = {
  "kind": "InlineFragment",
  "selections": (v5/*: any*/),
  "type": "Workspace",
  "abstractKey": null
},
v8 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "name",
  "storageKey": null
},
v9 = [
  (v3/*: any*/),
  (v8/*: any*/),
  (v4/*: any*/)
],
v10 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "system",
  "storageKey": null
},
v11 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "groupPath",
  "storageKey": null
},
v12 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "registryNamespace",
  "storageKey": null
},
v13 = [
  (v8/*: any*/),
  (v4/*: any*/)
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "UniversalSearchQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "SearchResponse",
        "kind": "LinkedField",
        "name": "search",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": null,
            "kind": "LinkedField",
            "name": "results",
            "plural": true,
            "selections": [
              (v2/*: any*/),
              {
                "kind": "InlineFragment",
                "selections": [
                  (v3/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": null,
                    "kind": "LinkedField",
                    "name": "namespace",
                    "plural": false,
                    "selections": [
                      (v2/*: any*/),
                      (v6/*: any*/),
                      (v7/*: any*/)
                    ],
                    "storageKey": null
                  }
                ],
                "type": "NamespaceFavorite",
                "abstractKey": null
              },
              {
                "kind": "InlineFragment",
                "selections": (v9/*: any*/),
                "type": "Group",
                "abstractKey": null
              },
              {
                "kind": "InlineFragment",
                "selections": (v9/*: any*/),
                "type": "Workspace",
                "abstractKey": null
              },
              {
                "kind": "InlineFragment",
                "selections": [
                  (v3/*: any*/),
                  (v8/*: any*/),
                  (v10/*: any*/),
                  (v11/*: any*/),
                  (v12/*: any*/)
                ],
                "type": "TerraformModule",
                "abstractKey": null
              },
              {
                "kind": "InlineFragment",
                "selections": [
                  (v3/*: any*/),
                  (v8/*: any*/),
                  (v11/*: any*/),
                  (v12/*: any*/)
                ],
                "type": "TerraformProvider",
                "abstractKey": null
              },
              {
                "kind": "InlineFragment",
                "selections": [
                  (v3/*: any*/),
                  (v8/*: any*/)
                ],
                "type": "Team",
                "abstractKey": null
              }
            ],
            "storageKey": null
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
    "name": "UniversalSearchQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "SearchResponse",
        "kind": "LinkedField",
        "name": "search",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": null,
            "kind": "LinkedField",
            "name": "results",
            "plural": true,
            "selections": [
              (v2/*: any*/),
              (v3/*: any*/),
              {
                "kind": "InlineFragment",
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": null,
                    "kind": "LinkedField",
                    "name": "namespace",
                    "plural": false,
                    "selections": [
                      (v2/*: any*/),
                      (v6/*: any*/),
                      (v7/*: any*/),
                      (v3/*: any*/)
                    ],
                    "storageKey": null
                  }
                ],
                "type": "NamespaceFavorite",
                "abstractKey": null
              },
              {
                "kind": "InlineFragment",
                "selections": (v13/*: any*/),
                "type": "Group",
                "abstractKey": null
              },
              {
                "kind": "InlineFragment",
                "selections": (v13/*: any*/),
                "type": "Workspace",
                "abstractKey": null
              },
              {
                "kind": "InlineFragment",
                "selections": [
                  (v8/*: any*/),
                  (v10/*: any*/),
                  (v11/*: any*/),
                  (v12/*: any*/)
                ],
                "type": "TerraformModule",
                "abstractKey": null
              },
              {
                "kind": "InlineFragment",
                "selections": [
                  (v8/*: any*/),
                  (v11/*: any*/),
                  (v12/*: any*/)
                ],
                "type": "TerraformProvider",
                "abstractKey": null
              },
              {
                "kind": "InlineFragment",
                "selections": [
                  (v8/*: any*/)
                ],
                "type": "Team",
                "abstractKey": null
              }
            ],
            "storageKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "d10d1561b7c22d1ea9b6ec79e7c4e196",
    "id": null,
    "metadata": {},
    "name": "UniversalSearchQuery",
    "operationKind": "query",
    "text": "query UniversalSearchQuery(\n  $query: String!\n) {\n  search(query: $query) {\n    results {\n      __typename\n      ... on NamespaceFavorite {\n        id\n        namespace {\n          __typename\n          ... on Group {\n            fullPath\n          }\n          ... on Workspace {\n            fullPath\n          }\n          id\n        }\n      }\n      ... on Group {\n        id\n        name\n        fullPath\n      }\n      ... on Workspace {\n        id\n        name\n        fullPath\n      }\n      ... on TerraformModule {\n        id\n        name\n        system\n        groupPath\n        registryNamespace\n      }\n      ... on TerraformProvider {\n        id\n        name\n        groupPath\n        registryNamespace\n      }\n      ... on Team {\n        id\n        name\n      }\n      id\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "049e07616ba35a2a591c7dfbf62b27a0";

export default node;
