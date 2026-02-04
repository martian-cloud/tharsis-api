/**
 * @generated SignedSource<<d4317ee28a30fd4891dd2603d944223f>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ProviderMirrorListFragment_mirrors$data = {
  readonly namespace: {
    readonly __typename: string;
    readonly id: string;
    readonly providerMirrorEnabled: {
      readonly value: boolean;
    };
    readonly terraformProviderMirrors: {
      readonly edges: ReadonlyArray<{
        readonly node: {
          readonly groupPath: string;
          readonly id: string;
          readonly providerAddress: string;
          readonly version: string;
          readonly " $fragmentSpreads": FragmentRefs<"ProviderMirrorListItemFragment_mirror">;
        } | null | undefined;
      } | null | undefined> | null | undefined;
      readonly totalCount: number;
    };
  } | null | undefined;
  readonly " $fragmentType": "ProviderMirrorListFragment_mirrors";
};
export type ProviderMirrorListFragment_mirrors$key = {
  readonly " $data"?: ProviderMirrorListFragment_mirrors$data;
  readonly " $fragmentSpreads": FragmentRefs<"ProviderMirrorListFragment_mirrors">;
};

import ProviderMirrorListPaginationQuery_graphql from './ProviderMirrorListPaginationQuery.graphql';

const node: ReaderFragment = (function(){
var v0 = [
  "namespace",
  "terraformProviderMirrors"
],
v1 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "__typename",
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
      "name": "namespacePath"
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
      "fragmentPathInResult": [],
      "operation": ProviderMirrorListPaginationQuery_graphql
    }
  },
  "name": "ProviderMirrorListFragment_mirrors",
  "selections": [
    {
      "alias": null,
      "args": [
        {
          "kind": "Variable",
          "name": "fullPath",
          "variableName": "namespacePath"
        }
      ],
      "concreteType": null,
      "kind": "LinkedField",
      "name": "namespace",
      "plural": false,
      "selections": [
        (v1/*: any*/),
        (v2/*: any*/),
        {
          "alias": null,
          "args": null,
          "concreteType": "NamespaceProviderMirrorEnabled",
          "kind": "LinkedField",
          "name": "providerMirrorEnabled",
          "plural": false,
          "selections": [
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "value",
              "storageKey": null
            }
          ],
          "storageKey": null
        },
        {
          "alias": "terraformProviderMirrors",
          "args": [
            {
              "kind": "Variable",
              "name": "search",
              "variableName": "search"
            },
            {
              "kind": "Literal",
              "name": "sort",
              "value": "TYPE_ASC"
            }
          ],
          "concreteType": "TerraformProviderVersionMirrorConnection",
          "kind": "LinkedField",
          "name": "__ProviderMirrorList_terraformProviderMirrors_connection",
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
              "concreteType": "TerraformProviderVersionMirrorEdge",
              "kind": "LinkedField",
              "name": "edges",
              "plural": true,
              "selections": [
                {
                  "alias": null,
                  "args": null,
                  "concreteType": "TerraformProviderVersionMirror",
                  "kind": "LinkedField",
                  "name": "node",
                  "plural": false,
                  "selections": [
                    (v1/*: any*/),
                    {
                      "alias": null,
                      "args": null,
                      "kind": "ScalarField",
                      "name": "version",
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
                      "name": "providerAddress",
                      "storageKey": null
                    },
                    {
                      "args": null,
                      "kind": "FragmentSpread",
                      "name": "ProviderMirrorListItemFragment_mirror"
                    },
                    (v2/*: any*/)
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

(node as any).hash = "25ea3adc6ad5e949775a661fb155ace1";

export default node;
