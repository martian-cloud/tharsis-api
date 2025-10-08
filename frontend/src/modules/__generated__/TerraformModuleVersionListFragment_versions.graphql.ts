/**
 * @generated SignedSource<<c07d0a35f0f656cfce8221d2d53a06f6>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TerraformModuleVersionListFragment_versions$data = {
  readonly node: {
    readonly versions?: {
      readonly edges: ReadonlyArray<{
        readonly node: {
          readonly id: string;
          readonly " $fragmentSpreads": FragmentRefs<"TerraformModuleVersionListItemFragment_version">;
        } | null | undefined;
      } | null | undefined> | null | undefined;
      readonly totalCount: number;
    };
  } | null | undefined;
  readonly " $fragmentType": "TerraformModuleVersionListFragment_versions";
};
export type TerraformModuleVersionListFragment_versions$key = {
  readonly " $data"?: TerraformModuleVersionListFragment_versions$data;
  readonly " $fragmentSpreads": FragmentRefs<"TerraformModuleVersionListFragment_versions">;
};

import TerraformModuleVersionListPaginationQuery_graphql from './TerraformModuleVersionListPaginationQuery.graphql';

const node: ReaderFragment = (function(){
var v0 = [
  "node",
  "versions"
];
return {
  "argumentDefinitions": [
    {
      "kind": "RootArgument",
      "name": "after"
    },
    {
      "kind": "RootArgument",
      "name": "before"
    },
    {
      "kind": "RootArgument",
      "name": "first"
    },
    {
      "kind": "RootArgument",
      "name": "last"
    },
    {
      "kind": "RootArgument",
      "name": "moduleId"
    }
  ],
  "kind": "Fragment",
  "metadata": {
    "connection": [
      {
        "count": null,
        "cursor": null,
        "direction": "bidirectional",
        "path": (v0/*: any*/)
      }
    ],
    "refetch": {
      "connection": {
        "forward": {
          "count": "first",
          "cursor": "after"
        },
        "backward": {
          "count": "last",
          "cursor": "before"
        },
        "path": (v0/*: any*/)
      },
      "fragmentPathInResult": [],
      "operation": TerraformModuleVersionListPaginationQuery_graphql
    }
  },
  "name": "TerraformModuleVersionListFragment_versions",
  "selections": [
    {
      "alias": null,
      "args": [
        {
          "kind": "Variable",
          "name": "id",
          "variableName": "moduleId"
        }
      ],
      "concreteType": null,
      "kind": "LinkedField",
      "name": "node",
      "plural": false,
      "selections": [
        {
          "kind": "InlineFragment",
          "selections": [
            {
              "alias": "versions",
              "args": [
                {
                  "kind": "Literal",
                  "name": "sort",
                  "value": "CREATED_AT_DESC"
                }
              ],
              "concreteType": "TerraformModuleVersionConnection",
              "kind": "LinkedField",
              "name": "__TerraformModuleVersionList_versions_connection",
              "plural": false,
              "selections": [
                {
                  "alias": null,
                  "args": null,
                  "kind": "ScalarField",
                  "name": "totalCount",
                  "storageKey": null
                },
                {
                  "alias": null,
                  "args": null,
                  "concreteType": "TerraformModuleVersionEdge",
                  "kind": "LinkedField",
                  "name": "edges",
                  "plural": true,
                  "selections": [
                    {
                      "alias": null,
                      "args": null,
                      "concreteType": "TerraformModuleVersion",
                      "kind": "LinkedField",
                      "name": "node",
                      "plural": false,
                      "selections": [
                        {
                          "alias": null,
                          "args": null,
                          "kind": "ScalarField",
                          "name": "id",
                          "storageKey": null
                        },
                        {
                          "args": null,
                          "kind": "FragmentSpread",
                          "name": "TerraformModuleVersionListItemFragment_version"
                        },
                        {
                          "alias": null,
                          "args": null,
                          "kind": "ScalarField",
                          "name": "__typename",
                          "storageKey": null
                        }
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
              "storageKey": "__TerraformModuleVersionList_versions_connection(sort:\"CREATED_AT_DESC\")"
            }
          ],
          "type": "TerraformModule",
          "abstractKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Query",
  "abstractKey": null
};
})();

(node as any).hash = "316233c0fdecf4321857040ae758068b";

export default node;
