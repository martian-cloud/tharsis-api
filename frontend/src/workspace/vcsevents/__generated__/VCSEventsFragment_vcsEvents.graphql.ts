/**
 * @generated SignedSource<<0f07578773eb32a6f38c74831289b656>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type VCSProviderType = "github" | "gitlab" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type VCSEventsFragment_vcsEvents$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly workspaceVcsProviderLink: {
    readonly id: string;
    readonly vcsProvider: {
      readonly type: VCSProviderType;
    };
  } | null | undefined;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceVCSWebhookAlertFragment_workspace">;
  readonly " $fragmentType": "VCSEventsFragment_vcsEvents";
};
export type VCSEventsFragment_vcsEvents$key = {
  readonly " $data"?: VCSEventsFragment_vcsEvents$data;
  readonly " $fragmentSpreads": FragmentRefs<"VCSEventsFragment_vcsEvents">;
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
  "name": "VCSEventsFragment_vcsEvents",
  "selections": [
    (v0/*:: as any*/),
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
        (v0/*:: as any*/),
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
      "name": "WorkspaceVCSWebhookAlertFragment_workspace"
    }
  ],
  "type": "Workspace",
  "abstractKey": null
};
})();

(node as any).hash = "289afca41f9d0ed71d4591196c983f4d";

export default node;
