/**
 * @generated SignedSource<<c1b3b7632e0371cb6d7173d5f860d869>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type RunnerType = "group" | "shared" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type RunnerDetailsFragment_runner$data = {
  readonly assignedServiceAccounts: {
    readonly totalCount: number;
  };
  readonly createdBy: string;
  readonly description: string;
  readonly disabled: boolean;
  readonly id: string;
  readonly metadata: {
    readonly createdAt: any;
    readonly trn: string;
  };
  readonly name: string;
  readonly runUntaggedJobs: boolean;
  readonly sessions: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly active: boolean;
        readonly lastContacted: any;
      } | null | undefined;
    } | null | undefined> | null | undefined;
  };
  readonly tags: ReadonlyArray<string>;
  readonly type: RunnerType;
  readonly " $fragmentSpreads": FragmentRefs<"AssignedServiceAccountListFragment_runner">;
  readonly " $fragmentType": "RunnerDetailsFragment_runner";
};
export type RunnerDetailsFragment_runner$key = {
  readonly " $data"?: RunnerDetailsFragment_runner$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunnerDetailsFragment_runner">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "RunnerDetailsFragment_runner",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "id",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "name",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "type",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "disabled",
      "storageKey": null
    },
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
      "name": "createdBy",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "tags",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "runUntaggedJobs",
      "storageKey": null
    },
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
        },
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
    {
      "alias": null,
      "args": [
        {
          "kind": "Literal",
          "name": "first",
          "value": 0
        }
      ],
      "concreteType": "ServiceAccountConnection",
      "kind": "LinkedField",
      "name": "assignedServiceAccounts",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "totalCount",
          "storageKey": null
        }
      ],
      "storageKey": "assignedServiceAccounts(first:0)"
    },
    {
      "alias": null,
      "args": [
        {
          "kind": "Literal",
          "name": "first",
          "value": 1
        },
        {
          "kind": "Literal",
          "name": "sort",
          "value": "LAST_CONTACTED_AT_DESC"
        }
      ],
      "concreteType": "RunnerSessionConnection",
      "kind": "LinkedField",
      "name": "sessions",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "concreteType": "RunnerSessionEdge",
          "kind": "LinkedField",
          "name": "edges",
          "plural": true,
          "selections": [
            {
              "alias": null,
              "args": null,
              "concreteType": "RunnerSession",
              "kind": "LinkedField",
              "name": "node",
              "plural": false,
              "selections": [
                {
                  "alias": null,
                  "args": null,
                  "kind": "ScalarField",
                  "name": "active",
                  "storageKey": null
                },
                {
                  "alias": null,
                  "args": null,
                  "kind": "ScalarField",
                  "name": "lastContacted",
                  "storageKey": null
                }
              ],
              "storageKey": null
            }
          ],
          "storageKey": null
        }
      ],
      "storageKey": "sessions(first:1,sort:\"LAST_CONTACTED_AT_DESC\")"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "AssignedServiceAccountListFragment_runner"
    }
  ],
  "type": "Runner",
  "abstractKey": null
};

(node as any).hash = "8cd98303202d63952ab6659c13ebf62f";

export default node;
