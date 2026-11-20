/**
 * @generated SignedSource<<52e2b51f67946d663a4ddb4ed452963e>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type VCSEventListItemFragment_vcsEvent$data = {
  readonly commitId: string | null | undefined;
  readonly errorMessage: string | null | undefined;
  readonly id: string;
  readonly metadata: {
    readonly createdAt: any;
    readonly trn: string;
  };
  readonly repositoryUrl: string;
  readonly sourceReferenceName: string | null | undefined;
  readonly status: string;
  readonly type: string;
  readonly " $fragmentType": "VCSEventListItemFragment_vcsEvent";
};
export type VCSEventListItemFragment_vcsEvent$key = {
  readonly " $data"?: VCSEventListItemFragment_vcsEvent$data;
  readonly " $fragmentSpreads": FragmentRefs<"VCSEventListItemFragment_vcsEvent">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "VCSEventListItemFragment_vcsEvent",
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
      "name": "status",
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
      "name": "sourceReferenceName",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "commitId",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "errorMessage",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "repositoryUrl",
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
    }
  ],
  "type": "VCSEvent",
  "abstractKey": null
};

(node as any).hash = "bff875bee3f6c7807ed9ea28fb464a4a";

export default node;
