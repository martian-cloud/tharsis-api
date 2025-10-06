/**
 * @generated SignedSource<<4f76dcf1c00b701ebe08b8a68002ecc4>>
 * @lightSyntaxTransform
 * @nogrep
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
  } | null | undefined;
  readonly " $fragmentSpreads": FragmentRefs<"EditVCSProviderLinkFragment_workspace" | "NewVCSProviderLinkFragment_workspace">;
  readonly " $fragmentType": "WorkspaceVCSProviderSettingsFragment_workspace";
};
export type WorkspaceVCSProviderSettingsFragment_workspace$key = {
  readonly " $data"?: WorkspaceVCSProviderSettingsFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceVCSProviderSettingsFragment_workspace">;
};

const node: ReaderFragment = (function(){
var v0 = [
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "id",
    "storageKey": null
  }
];
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "WorkspaceVCSProviderSettingsFragment_workspace",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "WorkspaceVCSProviderLink",
      "kind": "LinkedField",
      "name": "workspaceVcsProviderLink",
      "plural": false,
      "selections": (v0/*: any*/),
      "storageKey": null
    },
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
      "args": [
        {
          "kind": "Literal",
          "name": "first",
          "value": 10
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
              "selections": (v0/*: any*/),
              "storageKey": null
            }
          ],
          "storageKey": null
        }
      ],
      "storageKey": "vcsProviders(first:10,includeInherited:true)"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "EditVCSProviderLinkFragment_workspace"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "NewVCSProviderLinkFragment_workspace"
    }
  ],
  "type": "Workspace",
  "abstractKey": null
};
})();

(node as any).hash = "322e818c25d81dc630957f0613e820dd";

export default node;
