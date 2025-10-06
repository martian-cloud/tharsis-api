/**
 * @generated SignedSource<<0f4ec699275f882fcb83caa39a75e019>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type VariableHistoryDialogFragment_variable$data = {
  readonly id: string;
  readonly versions: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly hcl: boolean | null | undefined;
        readonly id: string;
        readonly key: string;
        readonly metadata: {
          readonly createdAt: any;
        };
        readonly value?: string | null | undefined;
      } | null | undefined;
    } | null | undefined> | null | undefined;
    readonly totalCount: number;
  };
  readonly " $fragmentType": "VariableHistoryDialogFragment_variable";
};
export type VariableHistoryDialogFragment_variable$key = {
  readonly " $data"?: VariableHistoryDialogFragment_variable$data;
  readonly " $fragmentSpreads": FragmentRefs<"VariableHistoryDialogFragment_variable">;
};

import VariableHistoryDialogPaginationQuery_graphql from './VariableHistoryDialogPaginationQuery.graphql';

const node: ReaderFragment = (function(){
var v0 = [
  "versions"
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
      "name": "first"
    },
    {
      "kind": "RootArgument",
      "name": "includeValues"
    }
  ],
  "kind": "Fragment",
  "metadata": {
    "connection": [
      {
        "count": "first",
        "cursor": "after",
        "direction": "forward",
        "path": (v0/*: any*/)
      }
    ],
    "refetch": {
      "connection": {
        "forward": {
          "count": "first",
          "cursor": "after"
        },
        "backward": null,
        "path": (v0/*: any*/)
      },
      "fragmentPathInResult": [
        "node"
      ],
      "operation": VariableHistoryDialogPaginationQuery_graphql,
      "identifierInfo": {
        "identifierField": "id",
        "identifierQueryVariableName": "id"
      }
    }
  },
  "name": "VariableHistoryDialogFragment_variable",
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
      "concreteType": "NamespaceVariableVersionConnection",
      "kind": "LinkedField",
      "name": "__VariableHistoryDialog_versions_connection",
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
          "concreteType": "NamespaceVariableVersionEdge",
          "kind": "LinkedField",
          "name": "edges",
          "plural": true,
          "selections": [
            {
              "alias": null,
              "args": null,
              "concreteType": "NamespaceVariableVersion",
              "kind": "LinkedField",
              "name": "node",
              "plural": false,
              "selections": [
                {
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
                (v1/*: any*/),
                {
                  "alias": null,
                  "args": null,
                  "kind": "ScalarField",
                  "name": "key",
                  "storageKey": null
                },
                {
                  "condition": "includeValues",
                  "kind": "Condition",
                  "passingValue": true,
                  "selections": [
                    {
                      "alias": null,
                      "args": null,
                      "kind": "ScalarField",
                      "name": "value",
                      "storageKey": null
                    }
                  ]
                },
                {
                  "alias": null,
                  "args": null,
                  "kind": "ScalarField",
                  "name": "hcl",
                  "storageKey": null
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
            }
          ],
          "storageKey": null
        }
      ],
      "storageKey": "__VariableHistoryDialog_versions_connection(sort:\"CREATED_AT_DESC\")"
    },
    (v1/*: any*/)
  ],
  "type": "NamespaceVariable",
  "abstractKey": null
};
})();

(node as any).hash = "30d73859dd9bb1b08bd84de3a2651b30";

export default node;
