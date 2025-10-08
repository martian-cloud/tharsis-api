/**
 * @generated SignedSource<<f86b44571e51ba4e925ff4de0d526536>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type RunnerSessionListItemFragment$data = {
  readonly active: boolean;
  readonly errorCount: number;
  readonly id: string;
  readonly internal: boolean;
  readonly lastContacted: any;
  readonly metadata: {
    readonly updatedAt: any;
  };
  readonly " $fragmentType": "RunnerSessionListItemFragment";
};
export type RunnerSessionListItemFragment$key = {
  readonly " $data"?: RunnerSessionListItemFragment$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunnerSessionListItemFragment">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "RunnerSessionListItemFragment",
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
      "name": "lastContacted",
      "storageKey": null
    },
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
      "name": "internal",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "errorCount",
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
          "name": "updatedAt",
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "RunnerSession",
  "abstractKey": null
};

(node as any).hash = "0828330786faf3fbedb0cb70c65198d4";

export default node;
