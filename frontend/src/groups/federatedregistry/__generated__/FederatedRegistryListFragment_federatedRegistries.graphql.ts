/**
 * @generated SignedSource<<906aa46a809bece288587179e3522e85>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type FederatedRegistryListFragment_federatedRegistries$data = {
  readonly federatedRegistries: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly id: string;
        readonly " $fragmentSpreads": FragmentRefs<"FederatedRegistryListItemFragment_federatedRegistry">;
      } | null | undefined;
    } | null | undefined> | null | undefined;
    readonly totalCount: number;
  };
  readonly id: string;
  readonly " $fragmentType": "FederatedRegistryListFragment_federatedRegistries";
};
export type FederatedRegistryListFragment_federatedRegistries$key = {
  readonly " $data"?: FederatedRegistryListFragment_federatedRegistries$data;
  readonly " $fragmentSpreads": FragmentRefs<"FederatedRegistryListFragment_federatedRegistries">;
};

import FederatedRegistryListPaginationQuery_graphql from './FederatedRegistryListPaginationQuery.graphql';

const node: ReaderFragment = (function(){
var v0 = [
  "federatedRegistries"
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
      "operation": FederatedRegistryListPaginationQuery_graphql,
      "identifierInfo": {
        "identifierField": "id",
        "identifierQueryVariableName": "id"
      }
    }
  },
  "name": "FederatedRegistryListFragment_federatedRegistries",
  "selections": [
    {
      "alias": "federatedRegistries",
      "args": [
        {
          "kind": "Literal",
          "name": "sort",
          "value": "UPDATED_AT_DESC"
        }
      ],
      "concreteType": "FederatedRegistryConnection",
      "kind": "LinkedField",
      "name": "__FederatedRegistryList_federatedRegistries_connection",
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
          "concreteType": "FederatedRegistryEdge",
          "kind": "LinkedField",
          "name": "edges",
          "plural": true,
          "selections": [
            {
              "alias": null,
              "args": null,
              "concreteType": "FederatedRegistry",
              "kind": "LinkedField",
              "name": "node",
              "plural": false,
              "selections": [
                (v1/*: any*/),
                {
                  "args": null,
                  "kind": "FragmentSpread",
                  "name": "FederatedRegistryListItemFragment_federatedRegistry"
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
      "storageKey": "__FederatedRegistryList_federatedRegistries_connection(sort:\"UPDATED_AT_DESC\")"
    },
    (v1/*: any*/)
  ],
  "type": "Group",
  "abstractKey": null
};
})();

(node as any).hash = "18a51d04d409ff6eaf280e7fa6ca023f";

export default node;
