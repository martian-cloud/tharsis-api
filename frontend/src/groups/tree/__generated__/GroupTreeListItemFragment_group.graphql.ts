/**
 * @generated SignedSource<<1afc8a6070b230f62e7a0da023816ffc>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupTreeListItemFragment_group$data = {
  readonly descendentGroups: {
    readonly totalCount: number;
  };
  readonly description: string;
  readonly fullPath: string;
  readonly id: string;
  readonly metadata: {
    readonly updatedAt: any;
  };
  readonly name: string;
  readonly workspaces: {
    readonly totalCount: number;
  };
  readonly " $fragmentType": "GroupTreeListItemFragment_group";
};
export type GroupTreeListItemFragment_group$key = {
  readonly " $data"?: GroupTreeListItemFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"GroupTreeListItemFragment_group">;
};

const node: ReaderFragment = (function(){
var v0 = [
  {
    "kind": "Literal",
    "name": "first",
    "value": 0
  }
],
v1 = [
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "totalCount",
    "storageKey": null
  }
];
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "GroupTreeListItemFragment_group",
  "selections": [
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
          "name": "updatedAt",
          "storageKey": null
        }
      ],
      "storageKey": null
    },
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
      "name": "fullPath",
      "storageKey": null
    },
    {
      "alias": null,
      "args": (v0/*: any*/),
      "concreteType": "GroupConnection",
      "kind": "LinkedField",
      "name": "descendentGroups",
      "plural": false,
      "selections": (v1/*: any*/),
      "storageKey": "descendentGroups(first:0)"
    },
    {
      "alias": null,
      "args": (v0/*: any*/),
      "concreteType": "WorkspaceConnection",
      "kind": "LinkedField",
      "name": "workspaces",
      "plural": false,
      "selections": (v1/*: any*/),
      "storageKey": "workspaces(first:0)"
    }
  ],
  "type": "Group",
  "abstractKey": null
};
})();

(node as any).hash = "a151a6570dc8e523c10cc3e9f04a5a0d";

export default node;
