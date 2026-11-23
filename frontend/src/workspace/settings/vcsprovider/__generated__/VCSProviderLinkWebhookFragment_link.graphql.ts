/**
 * @generated SignedSource<<680bface06fcb81f9d6788961dca5fec>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type VCSProviderType = "github" | "gitlab" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type VCSProviderLinkWebhookFragment_link$data = {
  readonly id: string;
  readonly repositoryPath: string;
  readonly usesLegacyWebhookToken: boolean;
  readonly vcsProvider: {
    readonly autoCreateWebhooks: boolean;
    readonly type: VCSProviderType;
  };
  readonly webhookDisabled: boolean;
  readonly " $fragmentType": "VCSProviderLinkWebhookFragment_link";
};
export type VCSProviderLinkWebhookFragment_link$key = {
  readonly " $data"?: VCSProviderLinkWebhookFragment_link$data;
  readonly " $fragmentSpreads": FragmentRefs<"VCSProviderLinkWebhookFragment_link">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "VCSProviderLinkWebhookFragment_link",
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
      "kind": "ScalarField",
      "name": "webhookDisabled",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "usesLegacyWebhookToken",
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
          "name": "type",
          "storageKey": null
        },
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

(node as any).hash = "b3cdf6d0dd37ea7d58a026b15acc6734";

export default node;
