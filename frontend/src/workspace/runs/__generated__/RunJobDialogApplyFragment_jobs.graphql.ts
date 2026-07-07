/**
 * @generated SignedSource<<f0b28a526f5c85aa1f8fd9d62b594631>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type RunJobDialogApplyFragment_jobs$data = {
  readonly apply: {
    readonly jobs: {
      readonly edges: ReadonlyArray<{
        readonly node: {
          readonly id: string;
          readonly " $fragmentSpreads": FragmentRefs<"RunJobDialog_jobs">;
        } | null | undefined;
      } | null | undefined> | null | undefined;
    };
  } | null | undefined;
  readonly id: string;
  readonly " $fragmentType": "RunJobDialogApplyFragment_jobs";
};
export type RunJobDialogApplyFragment_jobs$key = {
  readonly " $data"?: RunJobDialogApplyFragment_jobs$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunJobDialogApplyFragment_jobs">;
};

import RunJobDialogApplyPaginationQuery_graphql from './RunJobDialogApplyPaginationQuery.graphql';

const node: ReaderFragment = (function(){
var v0 = [
  "apply",
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
      "operation": RunJobDialogApplyPaginationQuery_graphql,
      "identifierInfo": {
        "identifierField": "id",
        "identifierQueryVariableName": "id"
      }
    }
  },
  "name": "RunJobDialogApplyFragment_jobs",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "Apply",
      "kind": "LinkedField",
      "name": "apply",
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
          "name": "__RunJobDialogApply_jobs_connection",
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
          "storageKey": "__RunJobDialogApply_jobs_connection(sort:\"CREATED_AT_DESC\")"
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

(node as any).hash = "daede607bd37677afa49b683c11c559c";

export default node;
