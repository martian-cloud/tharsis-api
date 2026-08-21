/**
 * @generated SignedSource<<7a4acad754058dc12587c2468215c7bf>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type WorkspaceDetailsCurrentApplyRunFragment_workspace$data = {
  readonly currentApplyRun: {
    readonly id: string;
    readonly " $fragmentSpreads": FragmentRefs<"RunStageIconsFragment_run">;
  } | null | undefined;
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "WorkspaceDetailsCurrentApplyRunFragment_workspace";
};
export type WorkspaceDetailsCurrentApplyRunFragment_workspace$key = {
  readonly " $data"?: WorkspaceDetailsCurrentApplyRunFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceDetailsCurrentApplyRunFragment_workspace">;
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
  "name": "WorkspaceDetailsCurrentApplyRunFragment_workspace",
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
      "concreteType": "Run",
      "kind": "LinkedField",
      "name": "currentApplyRun",
      "plural": false,
      "selections": [
        (v0/*: any*/),
        {
          "args": null,
          "kind": "FragmentSpread",
          "name": "RunStageIconsFragment_run"
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Workspace",
  "abstractKey": null
};
})();

(node as any).hash = "5172d86bbbad207d1385efed38df436c";

export default node;
