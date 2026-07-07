/**
 * @generated SignedSource<<915cc9b98873134e27321b72c09eb587>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type RunJobDialogPlanFragment_jobs$data = {
  readonly id: string;
  readonly plan: {
    readonly jobs: {
      readonly edges: ReadonlyArray<{
        readonly node: {
          readonly id: string;
          readonly " $fragmentSpreads": FragmentRefs<"RunJobDialog_jobs">;
        } | null | undefined;
      } | null | undefined> | null | undefined;
    };
  };
  readonly " $fragmentType": "RunJobDialogPlanFragment_jobs";
};
export type RunJobDialogPlanFragment_jobs$key = {
  readonly " $data"?: RunJobDialogPlanFragment_jobs$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunJobDialogPlanFragment_jobs">;
};

import RunJobDialogPlanPaginationQuery_graphql from './RunJobDialogPlanPaginationQuery.graphql';

const node: ReaderFragment = (function(){
var v0 = [
  "plan",
  "jobs"
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
      "operation": RunJobDialogPlanPaginationQuery_graphql,
      "identifierInfo": {
        "identifierField": "id",
        "identifierQueryVariableName": "id"
      }
    }
  },
  "name": "RunJobDialogPlanFragment_jobs",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "Plan",
      "kind": "LinkedField",
      "name": "plan",
      "plural": false,
      "selections": [
        {
          "alias": "jobs",
          "args": [
            {
              "kind": "Literal",
              "name": "sort",
              "value": "CREATED_AT_DESC"
            }
          ],
          "concreteType": "JobConnection",
          "kind": "LinkedField",
          "name": "__RunJobDialogPlan_jobs_connection",
          "plural": false,
          "selections": [
            {
              "alias": null,
              "args": null,
              "concreteType": "JobEdge",
              "kind": "LinkedField",
              "name": "edges",
              "plural": true,
              "selections": [
                {
                  "alias": null,
                  "args": null,
                  "concreteType": "Job",
                  "kind": "LinkedField",
                  "name": "node",
                  "plural": false,
                  "selections": [
                    (v1/*: any*/),
                    {
                      "args": null,
                      "kind": "FragmentSpread",
                      "name": "RunJobDialog_jobs"
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
          "storageKey": "__RunJobDialogPlan_jobs_connection(sort:\"CREATED_AT_DESC\")"
        }
      ],
      "storageKey": null
    },
    (v1/*: any*/)
  ],
  "type": "Run",
  "abstractKey": null
};
})();

(node as any).hash = "a417283034060e93adb5fc6a40dc01b4";

export default node;
