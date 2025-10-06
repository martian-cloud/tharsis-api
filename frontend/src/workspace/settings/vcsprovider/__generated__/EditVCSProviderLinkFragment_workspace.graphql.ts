/**
 * @generated SignedSource<<e8687162fe0a457dd1dbb5a8cffe6949>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type VCSProviderType = "github" | "gitlab" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type EditVCSProviderLinkFragment_workspace$data = {
  readonly fullPath: string;
  readonly workspaceVcsProviderLink: {
    readonly autoSpeculativePlan: boolean;
    readonly branch: string;
    readonly createdBy: string;
    readonly globPatterns: ReadonlyArray<string>;
    readonly id: string;
    readonly metadata: {
      readonly createdAt: any;
    };
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
  readonly " $fragmentSpreads": FragmentRefs<"VCSProviderLinkFormFragment_workspace">;
  readonly " $fragmentType": "EditVCSProviderLinkFragment_workspace";
};
export type EditVCSProviderLinkFragment_workspace$key = {
  readonly " $data"?: EditVCSProviderLinkFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"EditVCSProviderLinkFragment_workspace">;
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
  "name": "EditVCSProviderLinkFragment_workspace",
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
            }
          ],
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
          "name": "repositoryPath",
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
          "kind": "ScalarField",
          "name": "moduleDirectory",
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
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "VCSProviderLinkFormFragment_workspace"
    }
  ],
  "type": "Workspace",
  "abstractKey": null
};
})();

(node as any).hash = "d1705e81e31f35228d7448af8b42c0aa";

export default node;
