/**
 * @generated SignedSource<<383aef3e6346081bb8d71a5b43fe51ed>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type WorkspaceVCSProviderSettingsFragment_workspace$data = {
  readonly fullPath: string;
  readonly groupPath: string;
  readonly vcsProviders: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly id: string;
      } | null | undefined;
    } | null | undefined> | null | undefined;
  };
  readonly workspaceVcsProviderLink: {
    readonly id: string;
    readonly " $fragmentSpreads": FragmentRefs<"EditVCSProviderLinkFragment_link" | "VCSProviderLinkDangerZoneFragment_link" | "VCSProviderLinkWebhookFragment_link">;
  } | null | undefined;
  readonly " $fragmentType": "WorkspaceVCSProviderSettingsFragment_workspace";
};
export type WorkspaceVCSProviderSettingsFragment_workspace$key = {
  readonly " $data"?: WorkspaceVCSProviderSettingsFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceVCSProviderSettingsFragment_workspace">;
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
  "name": "WorkspaceVCSProviderSettingsFragment_workspace",
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
      "kind": "ScalarField",
      "name": "groupPath",
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
          "args": null,
          "kind": "FragmentSpread",
          "name": "EditVCSProviderLinkFragment_link"
        },
        {
          "args": null,
          "kind": "FragmentSpread",
          "name": "VCSProviderLinkWebhookFragment_link"
        },
        {
          "args": null,
          "kind": "FragmentSpread",
          "name": "VCSProviderLinkDangerZoneFragment_link"
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
          "value": 1
        },
        {
          "kind": "Literal",
          "name": "includeInherited",
          "value": true
        }
      ],
      "concreteType": "VCSProviderConnection",
      "kind": "LinkedField",
      "name": "vcsProviders",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "concreteType": "VCSProviderEdge",
          "kind": "LinkedField",
          "name": "edges",
          "plural": true,
          "selections": [
            {
              "alias": null,
              "args": null,
              "concreteType": "VCSProvider",
              "kind": "LinkedField",
              "name": "node",
              "plural": false,
              "selections": [
                (v0/*:: as any*/)
              ],
              "storageKey": null
            }
          ],
          "storageKey": null
        }
      ],
      "storageKey": "vcsProviders(first:1,includeInherited:true)"
    }
  ],
  "type": "Workspace",
  "abstractKey": null
};
})();

(node as any).hash = "811f0d274bcf3aa569eee4510a54b419";

export default node;
