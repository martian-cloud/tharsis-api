/**
 * @generated SignedSource<<3a612ab2816626634cdaae5a19b322ce>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type VCSProviderLinkDangerZoneFragment_link$data = {
  readonly id: string;
  readonly repositoryPath: string;
  readonly vcsProvider: {
    readonly autoCreateWebhooks: boolean;
  };
  readonly " $fragmentType": "VCSProviderLinkDangerZoneFragment_link";
};
export type VCSProviderLinkDangerZoneFragment_link$key = {
  readonly " $data"?: VCSProviderLinkDangerZoneFragment_link$data;
  readonly " $fragmentSpreads": FragmentRefs<"VCSProviderLinkDangerZoneFragment_link">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "VCSProviderLinkDangerZoneFragment_link",
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
      "name": "repositoryPath",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "VCSProvider",
      "kind": "LinkedField",
      "name": "vcsProvider",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "autoCreateWebhooks",
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "WorkspaceVCSProviderLink",
  "abstractKey": null
};

(node as any).hash = "231d05d3d46a3cbfc046604b02836210";

export default node;
