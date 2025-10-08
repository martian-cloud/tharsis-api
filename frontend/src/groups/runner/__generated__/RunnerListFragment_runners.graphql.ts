/**
 * @generated SignedSource<<f1ed6d6355dc209a40393cac11d0d828>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment, RefetchableFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type RunnerListFragment_runners$data = {
  readonly group: {
    readonly runners: {
      readonly edges: ReadonlyArray<{
        readonly node: {
          readonly id: string;
          readonly " $fragmentSpreads": FragmentRefs<"RunnerListItemFragment_runner">;
        } | null;
      } | null> | null;
      readonly totalCount: number;
    };
  } | null;
  readonly " $fragmentType": "RunnerListFragment_runners";
};
export type RunnerListFragment_runners$key = {
  readonly " $data"?: RunnerListFragment_runners$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunnerListFragment_runners">;
};

const node: ReaderFragment = (function(){
var v0 = [
  "group",
  "runners"
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
      "name": "groupPath"
    },
    {
      "kind": "RootArgument",
      "name": "includeInherited"
    },
    {
      "kind": "RootArgument",
      "name": "last"
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
      "operation": require('./RunnerListPaginationQuery.graphql')
    }
  },
  "name": "RunnerListFragment_runners",
  "selections": [
    {
      "alias": null,
      "args": [
        {
          "kind": "Variable",
          "name": "fullPath",
          "variableName": "groupPath"
        }
      ],
      "concreteType": "Group",
      "kind": "LinkedField",
      "name": "group",
      "plural": false,
      "selections": [
        {
          "alias": "runners",
          "args": [
            {
              "kind": "Variable",
              "name": "includeInherited",
              "variableName": "includeInherited"
            }
          ],
          "concreteType": "RunnerConnection",
          "kind": "LinkedField",
          "name": "__RunnerList_runners_connection",
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
              "concreteType": "RunnerEdge",
              "kind": "LinkedField",
              "name": "edges",
              "plural": true,
              "selections": [
                {
                  "alias": null,
                  "args": null,
                  "concreteType": "Runner",
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
                      "name": "RunnerListItemFragment_runner"
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
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Query",
  "abstractKey": null
};
})();

(node as any).hash = "1a76238049efcda40d7ac38284ca61de";

export default node;
