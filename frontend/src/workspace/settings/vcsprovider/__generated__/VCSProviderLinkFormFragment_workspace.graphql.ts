/**
 * @generated SignedSource<<12ba1db267cb3e0a61f9bc2e0654a74e>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type VCSProviderType = "github" | "gitlab" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type VCSProviderLinkFormFragment_workspace$data = {
  readonly fullPath: string;
  readonly workspaceVcsProviderLink: {
    readonly autoSpeculativePlan: boolean;
    readonly branch: string;
    readonly globPatterns: ReadonlyArray<string>;
    readonly id: string;
    readonly moduleDirectory: string | null | undefined;
    readonly repositoryPath: string;
    readonly tagRegex: string | null | undefined;
    readonly vcsProvider: {
      readonly autoCreateWebhooks: boolean;
      readonly description: string;
      readonly id: string;
      readonly name: string;
      readonly type: VCSProviderType;
    };
    readonly webhookDisabled: boolean;
  } | null | undefined;
  readonly " $fragmentType": "VCSProviderLinkFormFragment_workspace";
};
export type VCSProviderLinkFormFragment_workspace$key = {
  readonly " $data"?: VCSProviderLinkFormFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"VCSProviderLinkFormFragment_workspace">;
};

const node: ReaderFragment = (function(){
var v0 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
};
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "VCSProviderLinkFormFragment_workspace",
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
        (v0/*: any*/),
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
          "name": "branch",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "moduleDirectory",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "tagRegex",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "globPatterns",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "autoSpeculativePlan",
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
          "concreteType": "VCSProvider",
          "kind": "LinkedField",
          "name": "vcsProvider",
          "plural": false,
          "selections": [
            (v0/*: any*/),
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
              "name": "description",
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
              "name": "autoCreateWebhooks",
              "storageKey": null
            }
          ],
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Workspace",
  "abstractKey": null
};
})();

(node as any).hash = "8312a5ba20092cf9e813883fc6c09c2e";

export default node;
