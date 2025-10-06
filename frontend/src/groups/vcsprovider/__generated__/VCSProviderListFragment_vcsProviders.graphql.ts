/**
 * @generated SignedSource<<145d083be28ea32f77874e36af17a02a>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type VCSProviderListFragment_vcsProviders$data = {
  readonly id: string;
  readonly vcsProviders: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly groupPath: string;
        readonly id: string;
        readonly " $fragmentSpreads": FragmentRefs<"VCSProviderListItemFragment_vcsProvider">;
      } | null | undefined;
    } | null | undefined> | null | undefined;
    readonly totalCount: number;
  };
  readonly " $fragmentType": "VCSProviderListFragment_vcsProviders";
};
export type VCSProviderListFragment_vcsProviders$key = {
  readonly " $data"?: VCSProviderListFragment_vcsProviders$data;
  readonly " $fragmentSpreads": FragmentRefs<"VCSProviderListFragment_vcsProviders">;
};

import VCSProviderListPaginationQuery_graphql from './VCSProviderListPaginationQuery.graphql';

const node: ReaderFragment = (function(){
var v0 = [
  "vcsProviders"
],
v1 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
};
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
      "name": "search"
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
      "fragmentPathInResult": [
        "node"
      ],
      "operation": VCSProviderListPaginationQuery_graphql,
      "identifierInfo": {
        "identifierField": "id",
        "identifierQueryVariableName": "id"
      }
    }
  },
  "name": "VCSProviderListFragment_vcsProviders",
  "selections": [
    {
      "alias": "vcsProviders",
      "args": [
        {
          "kind": "Literal",
          "name": "includeInherited",
          "value": true
        },
        {
          "kind": "Variable",
          "name": "search",
          "variableName": "search"
        },
        {
          "kind": "Literal",
          "name": "sort",
          "value": "GROUP_LEVEL_DESC"
        }
      ],
      "concreteType": "VCSProviderConnection",
      "kind": "LinkedField",
      "name": "__VCSProviderList_vcsProviders_connection",
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
          "concreteType": "VCSProviderEdge",
          "kind": "LinkedField",
          "name": "edges",
          "plural": true,
          "selections": [
            {
              "alias": null,
              "args": null,
              "concreteType": "VCSProvider",
              "kind": "LinkedField",
              "name": "node",
              "plural": false,
              "selections": [
                (v1/*: any*/),
                {
                  "alias": null,
                  "args": null,
                  "kind": "ScalarField",
                  "name": "groupPath",
                  "storageKey": null
                },
                {
                  "args": null,
                  "kind": "FragmentSpread",
                  "name": "VCSProviderListItemFragment_vcsProvider"
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
    },
    (v1/*: any*/)
  ],
  "type": "Group",
  "abstractKey": null
};
})();

(node as any).hash = "0cfc1d237ae8df3e74572cfdb715622a";

export default node;
