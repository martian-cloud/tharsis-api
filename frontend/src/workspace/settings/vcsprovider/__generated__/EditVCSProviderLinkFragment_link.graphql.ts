/**
 * @generated SignedSource<<fc45f956b4fc5f3c4b9c0c0f8c086a89>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type VCSProviderType = "github" | "gitlab" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type EditVCSProviderLinkFragment_link$data = {
  readonly autoSpeculativePlan: boolean;
  readonly branch: string;
  readonly globPatterns: ReadonlyArray<string>;
  readonly id: string;
  readonly moduleDirectory: string | null | undefined;
  readonly repositoryPath: string;
  readonly tagRegex: string | null | undefined;
  readonly vcsProvider: {
    readonly autoCreateWebhooks: boolean;
    readonly groupPath: string;
    readonly id: string;
    readonly name: string;
    readonly type: VCSProviderType;
  };
  readonly webhookDisabled: boolean;
  readonly " $fragmentType": "EditVCSProviderLinkFragment_link";
};
export type EditVCSProviderLinkFragment_link$key = {
  readonly " $data"?: EditVCSProviderLinkFragment_link$data;
  readonly " $fragmentSpreads": FragmentRefs<"EditVCSProviderLinkFragment_link">;
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
  "name": "EditVCSProviderLinkFragment_link",
  "selections": [
    (v0/*:: as any*/),
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
        (v0/*:: as any*/),
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
          "name": "groupPath",
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
  "type": "WorkspaceVCSProviderLink",
  "abstractKey": null
};
})();

(node as any).hash = "7dac3b30fa928aad8cc18140bcdc94e5";

export default node;
