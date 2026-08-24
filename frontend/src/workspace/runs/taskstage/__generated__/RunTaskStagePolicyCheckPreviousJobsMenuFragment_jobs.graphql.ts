/**
 * @generated SignedSource<<90c40e6ae65a2d23f715a22e12572f19>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type RunTaskStagePolicyCheckPreviousJobsMenuFragment_jobs$data = {
  readonly runNode: {
    readonly jobs?: {
      readonly edges: ReadonlyArray<{
        readonly node: {
          readonly id: string;
          readonly " $fragmentSpreads": FragmentRefs<"RunTaskStagePolicyCheckPreviousJobsMenu_job">;
        } | null | undefined;
      } | null | undefined> | null | undefined;
    };
  } | null | undefined;
  readonly " $fragmentType": "RunTaskStagePolicyCheckPreviousJobsMenuFragment_jobs";
};
export type RunTaskStagePolicyCheckPreviousJobsMenuFragment_jobs$key = {
  readonly " $data"?: RunTaskStagePolicyCheckPreviousJobsMenuFragment_jobs$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunTaskStagePolicyCheckPreviousJobsMenuFragment_jobs">;
};

import RunTaskStagePolicyCheckPreviousJobsMenuPaginationQuery_graphql from './RunTaskStagePolicyCheckPreviousJobsMenuPaginationQuery.graphql';

const node: ReaderFragment = (function(){
var v0 = [
  "runNode",
  "jobs"
];
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
      "name": "nodePath"
    },
    {
      "kind": "RootArgument",
      "name": "runId"
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
      "fragmentPathInResult": [],
      "operation": RunTaskStagePolicyCheckPreviousJobsMenuPaginationQuery_graphql
    }
  },
  "name": "RunTaskStagePolicyCheckPreviousJobsMenuFragment_jobs",
  "selections": [
    {
      "alias": null,
      "args": [
        {
          "kind": "Variable",
          "name": "nodePath",
          "variableName": "nodePath"
        },
        {
          "kind": "Variable",
          "name": "runId",
          "variableName": "runId"
        }
      ],
      "concreteType": null,
      "kind": "LinkedField",
      "name": "runNode",
      "plural": false,
      "selections": [
        {
          "kind": "InlineFragment",
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
              "name": "__RunTaskStagePolicyCheckPreviousJobsMenu_jobs_connection",
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
                          "name": "RunTaskStagePolicyCheckPreviousJobsMenu_job"
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
              "storageKey": "__RunTaskStagePolicyCheckPreviousJobsMenu_jobs_connection(sort:\"CREATED_AT_DESC\")"
            }
          ],
          "type": "PolicyCheck",
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

(node as any).hash = "8af478c43d0825a5e8b4f4a1a5252326";

export default node;
