/**
 * @generated SignedSource<<f95649066f4001ce71322f48944cc300>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type CreateRunFragment_workspace$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly workspaceVcsProviderLink: {
    readonly id: string;
  } | null | undefined;
  readonly " $fragmentSpreads": FragmentRefs<"ModuleSourceFragment_workspace" | "VCSWorkspaceLinkSourceFragment_workspace">;
  readonly " $fragmentType": "CreateRunFragment_workspace";
};
export type CreateRunFragment_workspace$key = {
  readonly " $data"?: CreateRunFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"CreateRunFragment_workspace">;
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
  "name": "CreateRunFragment_workspace",
  "selections": [
    (v0/*: any*/),
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
        (v0/*: any*/)
      ],
      "storageKey": null
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "ModuleSourceFragment_workspace"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "VCSWorkspaceLinkSourceFragment_workspace"
    }
  ],
  "type": "Workspace",
  "abstractKey": null
};
})();

(node as any).hash = "a953ec0b7a770d26e2e69220cb02250a";

export default node;
