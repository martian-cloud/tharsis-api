/**
 * @generated SignedSource<<8f6e50ed6951de8275b7c69bfaa78ee1>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type WorkspaceVCSWebhookAlertFragment_workspace$data = {
  readonly fullPath: string;
  readonly workspaceVcsProviderLink: {
    readonly usesLegacyWebhookToken: boolean;
    readonly webhookDisabled: boolean;
  } | null | undefined;
  readonly " $fragmentType": "WorkspaceVCSWebhookAlertFragment_workspace";
};
export type WorkspaceVCSWebhookAlertFragment_workspace$key = {
  readonly " $data"?: WorkspaceVCSWebhookAlertFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceVCSWebhookAlertFragment_workspace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "WorkspaceVCSWebhookAlertFragment_workspace",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "fullPath",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "WorkspaceVCSProviderLink",
      "kind": "LinkedField",
      "name": "workspaceVcsProviderLink",
      "plural": false,
      "selections": [
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
          "kind": "ScalarField",
          "name": "webhookDisabled",
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Workspace",
  "abstractKey": null
};

(node as any).hash = "68c65de017855d8fa23be6c7cca59d85";

export default node;
