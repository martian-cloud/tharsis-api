/**
 * @generated SignedSource<<b1f260241a1902a1a872f60fb717f705>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type RegistryPackageVersionDetailsQuery$variables = {
  after?: string | null | undefined;
  before?: string | null | undefined;
  first?: number | null | undefined;
  id: string;
  last?: number | null | undefined;
};
export type RegistryPackageVersionDetailsQuery$data = {
  readonly node: {
    readonly id?: string;
    readonly metadata?: {
      readonly trn: string;
    };
    readonly name?: string;
    readonly " $fragmentSpreads": FragmentRefs<"PackageDetailsIndexFragment_package">;
  } | null | undefined;
};
export type RegistryPackageVersionDetailsQuery = {
  response: RegistryPackageVersionDetailsQuery$data;
  variables: RegistryPackageVersionDetailsQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "after"
},
v1 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "before"
},
v2 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "first"
},
v3 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "id"
},
v4 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "last"
},
v5 = [
  {
    "kind": "Variable",
    "name": "id",
    "variableName": "id"
  }
],
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "name",
  "storageKey": null
},
v8 = {
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
      "name": "trn",
      "storageKey": null
    }
  ],
  "storageKey": null
},
v9 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "__typename",
  "storageKey": null
},
v10 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "version",
  "storageKey": null
},
v11 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "status",
  "storageKey": null
},
v12 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "latest",
  "storageKey": null
},
v13 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "createdBy",
  "storageKey": null
},
v14 = {
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
      "name": "createdAt",
      "storageKey": null
    }
  ],
  "storageKey": null
},
v15 = [
  {
    "kind": "Variable",
    "name": "after",
    "variableName": "after"
  },
  {
    "kind": "Variable",
    "name": "before",
    "variableName": "before"
  },
  {
    "kind": "Variable",
    "name": "first",
    "variableName": "first"
  },
  {
    "kind": "Variable",
    "name": "last",
    "variableName": "last"
  },
  {
    "kind": "Literal",
    "name": "sort",
    "value": "CREATED_AT_DESC"
  }
];
return {
  "fragment": {
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/),
      (v2/*: any*/),
      (v3/*: any*/),
      (v4/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "RegistryPackageVersionDetailsQuery",
    "selections": [
      {
        "alias": null,
        "args": (v5/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          {
            "kind": "InlineFragment",
            "selections": [
              (v6/*: any*/),
              (v7/*: any*/),
              (v8/*: any*/),
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "PackageDetailsIndexFragment_package"
              }
            ],
            "type": "Package",
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
    "argumentDefinitions": [
      (v3/*: any*/),
      (v2/*: any*/),
      (v4/*: any*/),
      (v0/*: any*/),
      (v1/*: any*/)
    ],
    "kind": "Operation",
    "name": "RegistryPackageVersionDetailsQuery",
    "selections": [
      {
        "alias": null,
        "args": (v5/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          (v9/*: any*/),
          (v6/*: any*/),
          {
            "kind": "InlineFragment",
            "selections": [
              (v7/*: any*/),
              (v8/*: any*/),
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
                "name": "groupPath",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "allowMutableVersions",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "PackageVersion",
                "kind": "LinkedField",
                "name": "latestVersion",
                "plural": false,
                "selections": [
                  (v6/*: any*/),
                  (v10/*: any*/),
                  (v11/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "error",
                    "storageKey": null
                  },
                  (v12/*: any*/),
                  (v13/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "shaSum",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "size",
                    "storageKey": null
                  },
                  (v14/*: any*/)
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "visibility",
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
                "args": (v15/*: any*/),
                "concreteType": "PackageVersionConnection",
                "kind": "LinkedField",
                "name": "versions",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "PackageVersionEdge",
                    "kind": "LinkedField",
                    "name": "edges",
                    "plural": true,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "concreteType": "PackageVersion",
                        "kind": "LinkedField",
                        "name": "node",
                        "plural": false,
                        "selections": [
                          (v6/*: any*/),
                          (v14/*: any*/),
                          (v10/*: any*/),
                          (v13/*: any*/),
                          (v12/*: any*/),
                          (v11/*: any*/),
                          (v9/*: any*/)
                        ],
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "cursor",
                        "storageKey": null
                      }
                    ],
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "PageInfo",
                    "kind": "LinkedField",
                    "name": "pageInfo",
                    "plural": false,
                    "selections": [
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "endCursor",
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "hasNextPage",
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "hasPreviousPage",
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "startCursor",
                        "storageKey": null
                      }
                    ],
                    "storageKey": null
                  }
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": (v15/*: any*/),
                "filters": [
                  "sort"
                ],
                "handle": "connection",
                "key": "PackageVersionList_versions",
                "kind": "LinkedHandle",
                "name": "versions"
              }
            ],
            "type": "Package",
            "abstractKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "ec5f1861d7d10efbe5c0961d6e6b7b7d",
    "id": null,
    "metadata": {},
    "name": "RegistryPackageVersionDetailsQuery",
    "operationKind": "query",
    "text": "query RegistryPackageVersionDetailsQuery(\n  $id: String!\n  $first: Int\n  $last: Int\n  $after: String\n  $before: String\n) {\n  node(id: $id) {\n    __typename\n    ... on Package {\n      id\n      name\n      metadata {\n        trn\n      }\n      ...PackageDetailsIndexFragment_package\n    }\n    id\n  }\n}\n\nfragment PackageDetailsIndexFragment_package on Package {\n  id\n  name\n  description\n  groupPath\n  allowMutableVersions\n  latestVersion {\n    id\n    ...PackageDetailsIndexFragment_version\n  }\n  ...PackageDetailsSidebarFragment_package\n  ...PackageVersionListFragment_package\n}\n\nfragment PackageDetailsIndexFragment_version on PackageVersion {\n  id\n  version\n  status\n  error\n  latest\n  ...PackageDetailsSidebarFragment_version\n}\n\nfragment PackageDetailsSidebarFragment_package on Package {\n  visibility\n  kind\n  groupPath\n  allowMutableVersions\n}\n\nfragment PackageDetailsSidebarFragment_version on PackageVersion {\n  version\n  latest\n  createdBy\n  shaSum\n  size\n  metadata {\n    createdAt\n  }\n}\n\nfragment PackageVersionListFragment_package on Package {\n  id\n  versions(after: $after, before: $before, first: $first, last: $last, sort: CREATED_AT_DESC) {\n    edges {\n      node {\n        id\n        ...PackageVersionListItemFragment_version\n        __typename\n      }\n      cursor\n    }\n    pageInfo {\n      endCursor\n      hasNextPage\n      hasPreviousPage\n      startCursor\n    }\n  }\n}\n\nfragment PackageVersionListItemFragment_version on PackageVersion {\n  metadata {\n    createdAt\n  }\n  id\n  version\n  createdBy\n  latest\n  status\n}\n"
  }
};
})();

(node as any).hash = "2fee455778a2cd3c4fab1af405b72280";

export default node;
