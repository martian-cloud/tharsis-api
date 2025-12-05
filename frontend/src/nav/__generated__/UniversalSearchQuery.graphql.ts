/**
 * @generated SignedSource<<83dcdf5f81f630855e788f12a0363d4c>>
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
  "name": "name",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "fullPath",
  "storageKey": null
},
v6 = [
  (v3/*: any*/),
  (v4/*: any*/),
  (v5/*: any*/)
],
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "system",
  "storageKey": null
},
v8 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "groupPath",
  "storageKey": null
},
v9 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "registryNamespace",
  "storageKey": null
},
v10 = [
  (v4/*: any*/),
  (v5/*: any*/)
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
                "selections": (v6/*: any*/),
                "type": "Group",
                "abstractKey": null
              },
              {
                "kind": "InlineFragment",
                "selections": (v6/*: any*/),
                "type": "Workspace",
                "abstractKey": null
              },
              {
                "kind": "InlineFragment",
                "selections": [
                  (v3/*: any*/),
                  (v4/*: any*/),
                  (v7/*: any*/),
                  (v8/*: any*/),
                  (v9/*: any*/)
                ],
                "type": "TerraformModule",
                "abstractKey": null
              },
              {
                "kind": "InlineFragment",
                "selections": [
                  (v3/*: any*/),
                  (v4/*: any*/),
                  (v8/*: any*/),
                  (v9/*: any*/)
                ],
                "type": "TerraformProvider",
                "abstractKey": null
              },
              {
                "kind": "InlineFragment",
                "selections": [
                  (v3/*: any*/),
                  (v4/*: any*/)
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
                "selections": (v10/*: any*/),
                "type": "Group",
                "abstractKey": null
              },
              {
                "kind": "InlineFragment",
                "selections": (v10/*: any*/),
                "type": "Workspace",
                "abstractKey": null
              },
              {
                "kind": "InlineFragment",
                "selections": [
                  (v4/*: any*/),
                  (v7/*: any*/),
                  (v8/*: any*/),
                  (v9/*: any*/)
                ],
                "type": "TerraformModule",
                "abstractKey": null
              },
              {
                "kind": "InlineFragment",
                "selections": [
                  (v4/*: any*/),
                  (v8/*: any*/),
                  (v9/*: any*/)
                ],
                "type": "TerraformProvider",
                "abstractKey": null
              },
              {
                "kind": "InlineFragment",
                "selections": [
                  (v4/*: any*/)
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
    "cacheID": "ce130488e4af2241fe68608434531712",
    "id": null,
    "metadata": {},
    "name": "UniversalSearchQuery",
    "operationKind": "query",
    "text": "query UniversalSearchQuery(\n  $query: String!\n) {\n  search(query: $query) {\n    results {\n      __typename\n      ... on Group {\n        id\n        name\n        fullPath\n      }\n      ... on Workspace {\n        id\n        name\n        fullPath\n      }\n      ... on TerraformModule {\n        id\n        name\n        system\n        groupPath\n        registryNamespace\n      }\n      ... on TerraformProvider {\n        id\n        name\n        groupPath\n        registryNamespace\n      }\n      ... on Team {\n        id\n        name\n      }\n      id\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "918f70f0f088ed79abd7cb8b5922fe29";

export default node;
